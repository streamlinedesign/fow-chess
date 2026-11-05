package engine

import "errors"

// ErrNotSync is returned for a round in a game played in turns.
var ErrNotSync = errors.New("not a sync move game")

// MovesFor lists c's moves as if it were c's turn. In a Sync Move game both
// players have moves every round.
func (p *Position) MovesFor(c Color) []Move {
	q := *p
	q.Turn = c
	return q.LegalMoves()
}

// FindMoveFor looks up c's move given as UCI.
func (p *Position) FindMoveFor(c Color, uci string) (Move, bool) {
	q := *p
	q.Turn = c
	return q.FindMove(uci)
}

// IsPass reports whether pm stands for a player who had no move in a Sync
// Move round. Every round adds two entries to the history, White's and then
// Black's, so a ply's parity still says whose it is.
func (pm PlayedMove) IsPass() bool { return pm.Move == Move{} }

// PlayRound plays one Sync Move round: White's and Black's moves as UCI, ""
// for a side with no move. Both are checked against the position at the
// start of the round, then happen at once:
//
//   - both moving pieces leave their squares first, so a piece that moves
//     away dodges a capture aimed at it, and two pieces can swap squares;
//   - a piece that lands on a square the other side's piece stayed on takes it;
//   - two pieces landing on the same square knock each other out: both are
//     taken, with whatever stood there;
//   - last, a side whose king was taken loses, and if both kings were taken
//     it's a draw. With King of the Hill, one king on the hill wins and two draw.
//
// There's no en passant, and SAN gets a "*" for a piece knocked out that way.
func (g *Game) PlayRound(ucis [2]string) ([2]PlayedMove, error) {
	var played [2]PlayedMove
	if g.Over() {
		return played, ErrGameOver
	}
	if !g.pos.Sync {
		return played, ErrNotSync
	}
	start := g.pos
	type mover struct {
		from, to Square
		pc       Piece
		by       int // the side whose move it is, or -1 for a castling rook
	}
	var movers []mover
	quiet := true
	for c := range 2 {
		if ucis[c] == "" {
			continue
		}
		m, ok := start.FindMoveFor(Color(c), ucis[c])
		if !ok {
			return [2]PlayedMove{}, ErrIllegalMove
		}
		pc := start.Board[m.From]
		if pc.Kind() == Pawn {
			quiet = false
		}
		if m.Promo != NoKind {
			pc = NewPiece(Color(c), m.Promo)
		}
		movers = append(movers, mover{m.From, m.To, pc, c})
		if m.Is(FlagCastle) {
			rank := m.From.Rank()
			rook, to := SquareAt(7, rank), SquareAt(5, rank)
			if m.To.File() != 6 {
				rook, to = SquareAt(0, rank), SquareAt(3, rank)
			}
			movers = append(movers, mover{rook, to, start.Board[rook], -1})
		}
		played[c].Move = m
	}

	p := start
	for _, mv := range movers {
		p.Board[mv.from] = NoPiece
		p.Castling &^= castleMask(mv.from) | castleMask(mv.to)
	}
	arriving := map[Square][]int{}
	var order []Square
	for i, mv := range movers {
		if arriving[mv.to] == nil {
			order = append(order, mv.to)
		}
		arriving[mv.to] = append(arriving[mv.to], i)
	}
	bumped := [2]bool{}
	for _, sq := range order {
		idx := arriving[sq]
		stayed := p.Board[sq]
		if !stayed.IsEmpty() {
			quiet = false
		}
		if len(idx) == 1 {
			mv := movers[idx[0]]
			if mv.by >= 0 && !stayed.IsEmpty() {
				played[mv.by].Captured = stayed
			}
			p.Board[sq] = mv.pc
			continue
		}
		quiet = false
		for _, i := range idx {
			by := movers[i].by
			if by < 0 {
				continue
			}
			bumped[by] = true
			for _, j := range idx {
				if movers[j].pc.Color() != Color(by) {
					played[by].Captured = movers[j].pc // the piece it ran into
				}
			}
		}
		p.Board[sq] = NoPiece
	}
	p.EP, p.Turn = NoSquare, White
	p.Fullmove++
	p.Halfmove++
	if !quiet {
		p.Halfmove = 0
	}

	for c := range 2 {
		pm := &played[c]
		if ucis[c] == "" {
			continue
		}
		m := pm.Move
		m.Flags &^= FlagCapture
		if !pm.Captured.IsEmpty() && !bumped[c] {
			m.Flags |= FlagCapture
		}
		q := start
		q.Turn = Color(c)
		pm.SAN = q.SAN(m)
		if bumped[c] {
			pm.SAN += "*"
		}
	}
	g.pos = p
	g.history = append(g.history, played[White], played[Black])
	g.seen[p.repetitionKey()]++
	g.evaluateRound()
	return played, nil
}

// evaluateRound applies the end-of-game rules after a Sync Move round.
func (g *Game) evaluateRound() {
	p := &g.pos
	wk, bk := p.KingSquare(White), p.KingSquare(Black)
	switch {
	case wk == NoSquare && bk == NoSquare:
		g.outcome, g.method = Drawn, KingCaptured
	case wk == NoSquare:
		g.outcome, g.method = BlackWins, KingCaptured
	case bk == NoSquare:
		g.outcome, g.method = WhiteWins, KingCaptured
	case p.Hill && OnHill(wk) && OnHill(bk):
		g.outcome, g.method = Drawn, KingOfTheHill
	case p.Hill && OnHill(wk):
		g.outcome, g.method = WhiteWins, KingOfTheHill
	case p.Hill && OnHill(bk):
		g.outcome, g.method = BlackWins, KingOfTheHill
	case len(p.MovesFor(White)) == 0 && len(p.MovesFor(Black)) == 0:
		g.outcome, g.method = Drawn, Stalemate
	case p.insufficientMaterial():
		g.outcome, g.method = Drawn, InsufficientMaterial
	case g.seen[p.repetitionKey()] >= 3:
		g.outcome, g.method = Drawn, Repetition
	case p.Halfmove >= 100:
		g.outcome, g.method = Drawn, FiftyMoves
	}
}

// Flag ends the game because c ran out of time: a loss, unless the opponent
// has too little left to ever win, which is a draw. In Sync Move either side
// can run out; in turns it's always the side to move (TimeOut).
func (g *Game) Flag(c Color) {
	if g.Over() {
		return
	}
	if g.pos.cannotWin(c.Other()) {
		g.outcome, g.method = Drawn, TimeoutVsInsufficientMaterial
	} else {
		g.outcome, g.method = winFor(c.Other()), Timeout
	}
}

// Finish ends the game with a result the moves don't show, as saved: for
// replaying a Sync Move game that ended on time.
func (g *Game) Finish(o Outcome, m Method) {
	if !g.Over() {
		g.outcome, g.method = o, m
	}
}

// Tokens is the history as saved, one per ply, or in a Sync Move game one per
// round: "s:" and each side's move or "-", like "s:e2e4,e7e5".
func (g *Game) Tokens() []string {
	var out []string
	if !g.pos.Sync {
		for _, pm := range g.history {
			out = append(out, pm.Move.UCI())
		}
		return out
	}
	for i := 0; i+1 < len(g.history); i += 2 {
		tok := "s:"
		for c, pm := range g.history[i : i+2] {
			if c > 0 {
				tok += ","
			}
			if pm.IsPass() {
				tok += "-"
			} else {
				tok += pm.Move.UCI()
			}
		}
		out = append(out, tok)
	}
	return out
}

// PlayToken plays a saved token; see Tokens.
func (g *Game) PlayToken(tok string) error {
	if len(tok) > 2 && tok[:2] == "s:" {
		var ucis [2]string
		parts := splitComma(tok[2:])
		if len(parts) != 2 {
			return errors.New("bad round " + tok)
		}
		for c, s := range parts {
			if s != "-" {
				ucis[c] = s
			}
		}
		_, err := g.PlayRound(ucis)
		return err
	}
	_, err := g.PlayUCI(tok)
	return err
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
