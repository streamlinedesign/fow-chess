package four

import "errors"

// MovesFor lists c's moves as if it were c's turn: in a Sync Move game every
// player still in has moves every round.
func (p *Position) MovesFor(c Color) []Move {
	if p.Out[c] {
		return nil
	}
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

// ErrNotSync is returned for a round in a game played in turns.
var ErrNotSync = errors.New("not a sync move game")

// PlayRound plays one Sync Move round: each player's move, "" for a player
// who is out or has no move. Every move is checked against the position at
// the start of the round, then all of them happen at once:
//
//   - every moving piece leaves its square first, so a piece that moves away
//     dodges a capture aimed at it, and two pieces can swap squares;
//   - a piece that lands on a square another player's piece stayed on takes it;
//   - two or more pieces landing on the same square knock each other out: all
//     of them are taken, with whatever stood there;
//   - last, every player whose king was taken is out. If nobody is left the
//     game is a draw; if one player is left, they win. With King of the Hill,
//     one king on a centre square wins, and two or more draw.
func (g *Game) PlayRound(ucis [4]string) ([]Event, error) {
	if g.over {
		return nil, ErrGameOver
	}
	if !g.pos.Sync {
		return nil, ErrNotSync
	}
	start := g.pos
	type mover struct {
		by       Color
		from, to Square
		pc       Piece
		event    int // index into events, or -1 for a castling rook
	}
	var movers []mover
	var events []Event
	g.rounds++
	for _, c := range Colors {
		if ucis[c] == "" || start.Out[c] {
			continue
		}
		m, ok := start.FindMoveFor(c, ucis[c])
		if !ok {
			g.rounds--
			return nil, ErrIllegalMove
		}
		pc := start.Board[m.From]
		if m.Promo != NoKind {
			pc = NewPiece(c, m.Promo)
		}
		movers = append(movers, mover{by: c, from: m.From, to: m.To, pc: pc, event: len(events)})
		if m.Is(FlagCastle) {
			_, kr, qr := castleSquares(c)
			rook, to := kr, cell(c, 5, 0)
			if m.To != cell(c, 6, 0) {
				rook, to = qr, cell(c, 3, 0)
			}
			movers = append(movers, mover{by: c, from: rook, to: to, pc: start.Board[rook], event: -1})
		}
		events = append(events, Event{By: c, Move: m, Round: g.rounds})
	}

	p := start
	for _, mv := range movers {
		p.Board[mv.from] = NoPiece
		p.Castling &^= castleMask(mv.from) | castleMask(mv.to)
	}
	arriving := map[Square][]int{}
	for i, mv := range movers {
		arriving[mv.to] = append(arriving[mv.to], i)
	}
	var lost []Piece
	for sq, idx := range arriving {
		stayed := p.Board[sq]
		if len(idx) == 1 {
			mv := movers[idx[0]]
			if !stayed.IsEmpty() {
				lost = append(lost, stayed)
				if mv.event >= 0 {
					events[mv.event].Captured = stayed
				}
			}
			p.Board[sq] = mv.pc
			continue
		}
		if !stayed.IsEmpty() {
			lost = append(lost, stayed)
		}
		for _, i := range idx {
			lost = append(lost, movers[i].pc)
			if e := movers[i].event; e >= 0 {
				events[e].Bumped = true
			}
		}
		p.Board[sq] = NoPiece
	}
	p.EP, p.EPPawn = NoSquare, NoSquare

	for i := range events {
		e := &events[i]
		m := e.Move
		m.Flags &^= FlagCapture
		if !e.Captured.IsEmpty() {
			m.Flags |= FlagCapture
		}
		e.SAN = sanAt(&start, m)
		if e.Captured.Kind() == King {
			e.SAN += "#"
		}
		if e.Bumped {
			e.SAN += "*"
		}
	}
	g.pos = p
	g.history = append(g.history, events...)

	for _, pc := range lost {
		if pc.Kind() == King && !g.pos.Out[pc.Color()] {
			g.pos.Eliminate(pc.Color())
			g.fell[pc.Color()] = KingCaptured
		}
	}
	if g.pos.Hill {
		var on []Color
		for _, c := range Colors {
			if k := g.pos.KingSquare(c); !g.pos.Out[c] && k != NoSquare && InCentre(k) {
				on = append(on, c)
			}
		}
		switch {
		case len(on) == 1:
			g.end(on[0], KingOfTheHill)
			return events, nil
		case len(on) > 1:
			g.over, g.drawn, g.method = true, true, KingOfTheHill
			return events, nil
		}
	}
	g.evaluate(KingCaptured)
	return events, nil
}
