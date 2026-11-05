package four

import (
	"errors"
	"strings"
)

// Vision is what c sees in fog of war, the same way as in two-player games:
// each piece sees its square and every square it could move to; a pawn always
// sees one square ahead, two from its starting row, and with back pawns the
// square behind. A player holding a centre square under centre vision sees
// everything, and so does a player who is out.
func (p *Position) Vision(c Color) [NumSquares]bool {
	var seen [NumSquares]bool
	if p.Out[c] || p.Outpost(c) != NoSquare {
		for i := range seen {
			seen[i] = true
		}
		return seen
	}
	for i, pc := range p.Board {
		if !pc.Is(c) {
			continue
		}
		sq := Square(i)
		seen[sq] = true
		if pc.Kind() == Pawn {
			fx, fy := forward[c][0], forward[c][1]
			if ahead, ok := sq.Offset(fx, fy); ok {
				seen[ahead] = true
				if rankOf(c, sq) == 1 {
					if two, ok := ahead.Offset(fx, fy); ok {
						seen[two] = true
					}
				}
			}
		}
		for _, m := range p.PieceMoves(sq) {
			seen[m.To] = true
		}
	}
	return seen
}

// Outpost is the centre square holding c's piece that gives c the whole board
// under centre vision, or NoSquare.
func (p *Position) Outpost(c Color) Square {
	if p.CenterVision == CenterOff {
		return NoSquare
	}
	for _, sq := range Centre {
		pc := p.Board[sq]
		if pc.Is(c) && (p.CenterVision == CenterAny || pc.Kind() == Pawn) {
			return sq
		}
	}
	return NoSquare
}

// Method is how a player went out, or how the game ended.
type Method uint8

const (
	NoMethod Method = iota
	KingCaptured
	Resignation
	Timeout
	KingOfTheHill
	// Stuck ends a game in a draw when nobody left can move.
	Stuck
	// Abandoned is a draw agreed by everyone still in.
	Agreement
)

func (m Method) String() string {
	return [...]string{"", "capturing the king", "resignation", "time", "reaching the hill", "no moves", "agreement"}[m]
}

// Event is one entry in a game's history: a move, or a player leaving the game
// by resigning or running out of time.
type Event struct {
	By Color
	// Move is set for a move.
	Move Move
	// Out is Resignation or Timeout for a player who left without a move.
	Out Method
	// SAN is the move in long algebraic notation, like "Nf1-g3" or "e2xd3".
	SAN string
	// Captured is the piece the move took, or NoPiece.
	Captured Piece
	// Round is the Sync Move round the move was part of, counted from 1; 0
	// in a game played in turns.
	Round int
	// Bumped is set when the piece landed on the same square as another
	// player's piece in a Sync Move round, and both were taken.
	Bumped bool
}

// Token is the event as saved: the move's UCI, or "-r" for Red resigning and
// "~r" for Red running out of time.
func (e Event) Token() string {
	switch e.Out {
	case Resignation:
		return "-" + e.By.String()
	case Timeout:
		return "~" + e.By.String()
	}
	return e.Move.UCI()
}

// Game is a position plus its history and result.
type Game struct {
	pos     Position
	history []Event
	// setups are the players' own arrangements, when the game started from them.
	setups [4]string
	// rounds is how many Sync Move rounds have been played.
	rounds int
	// fell is how each player went out, NoMethod while they're in.
	fell   [4]Method
	over   bool
	winner Color
	drawn  bool
	method Method
}

var ErrGameOver = errors.New("game is over")
var ErrIllegalMove = errors.New("illegal move")

func NewGame(r Rules) *Game {
	return &Game{pos: StartPosition(r)}
}

func (g *Game) Position() Position { return g.pos }
func (g *Game) Turn() Color        { return g.pos.Turn }
func (g *Game) History() []Event   { return g.history }
func (g *Game) Over() bool         { return g.over }
func (g *Game) Method() Method     { return g.method }
func (g *Game) LegalMoves() []Move { return g.pos.LegalMoves() }

// Winner is who won, once the game is over and wasn't drawn.
func (g *Game) Winner() (Color, bool) { return g.winner, g.over && !g.drawn }

// Fell is how c went out, or NoMethod while c is still playing.
func (g *Game) Fell(c Color) Method { return g.fell[c] }

// Result is "r", "b", "g" or "v" for the winner, "draw", or "" while playing.
func (g *Game) Result() string {
	switch {
	case !g.over:
		return ""
	case g.drawn:
		return "draw"
	}
	return g.winner.String()
}

// PlayUCI plays the side to move's move if it's legal.
func (g *Game) PlayUCI(uci string) (Event, error) {
	if g.over {
		return Event{}, ErrGameOver
	}
	m, ok := g.pos.FindMove(uci)
	if !ok {
		return Event{}, ErrIllegalMove
	}
	us := g.pos.Turn
	san := g.san(m)
	next, captured := g.pos.Apply(m)
	g.pos = next
	if captured.Kind() == King {
		g.fell[captured.Color()] = KingCaptured
		san += "#"
	}
	e := Event{By: us, Move: m, SAN: san, Captured: captured}
	g.history = append(g.history, e)
	if g.pos.Hill && g.pos.Board[m.To].Kind() == King && InCentre(m.To) {
		g.end(us, KingOfTheHill)
		return e, nil
	}
	g.evaluate(KingCaptured)
	return e, nil
}

// Leave takes c out of the game for resigning or running out of time.
func (g *Game) Leave(c Color, why Method) {
	if g.over || g.pos.Out[c] {
		return
	}
	g.pos.Eliminate(c)
	g.fell[c] = why
	g.history = append(g.history, Event{By: c, Out: why})
	g.evaluate(why)
}

// AgreeDraw ends the game as a draw.
func (g *Game) AgreeDraw() {
	if !g.over {
		g.over, g.drawn, g.method = true, true, Agreement
	}
}

// PlayToken replays a saved event; see Event.Token.
func (g *Game) PlayToken(tok string) error {
	if rest, ok := strings.CutPrefix(tok, "s:"); ok {
		parts := strings.Split(rest, ",")
		if len(parts) != 4 {
			return errors.New("bad round " + tok)
		}
		var moves [4]string
		for i, p := range parts {
			if p != "-" {
				moves[i] = p
			}
		}
		_, err := g.PlayRound(moves)
		return err
	}
	if len(tok) == 2 && (tok[0] == '-' || tok[0] == '~') {
		c, ok := ParseColor(tok[1:])
		if !ok {
			return errors.New("bad event " + tok)
		}
		why := Resignation
		if tok[0] == '~' {
			why = Timeout
		}
		g.Leave(c, why)
		return nil
	}
	_, err := g.PlayUCI(tok)
	return err
}

// Tokens is the history as saved, space separated.
func (g *Game) Tokens() string {
	return strings.Join(g.TokenList(), " ")
}

// TokenList is the history as saved, one token per move, leaving, or Sync
// Move round ("s:" and each player's move or "-", like "s:e2e4,-,j13j11,m10k10").
func (g *Game) TokenList() []string {
	var out []string
	for i := 0; i < len(g.history); i++ {
		e := g.history[i]
		if e.Round == 0 {
			out = append(out, e.Token())
			continue
		}
		parts := [4]string{"-", "-", "-", "-"}
		for ; i < len(g.history) && g.history[i].Round == e.Round; i++ {
			parts[g.history[i].By] = g.history[i].Move.UCI()
		}
		i--
		out = append(out, "s:"+strings.Join(parts[:], ","))
	}
	return out
}

func (g *Game) end(winner Color, why Method) {
	g.over, g.winner, g.method = true, winner, why
}

// evaluate ends the game once one player is left, and passes the turn over a
// player who has no move. why is how the last player went out.
func (g *Game) evaluate(why Method) {
	p := &g.pos
	if p.Alive() == 1 {
		for _, c := range Colors {
			if !p.Out[c] {
				g.end(c, why)
			}
		}
		return
	}
	if p.Alive() == 0 {
		g.over, g.drawn, g.method = true, true, why
		return
	}
	if p.Sync {
		for _, c := range Colors {
			if !p.Out[c] && len(p.MovesFor(c)) > 0 {
				return
			}
		}
		g.over, g.drawn, g.method = true, true, Stuck
		return
	}
	for range 4 {
		if len(p.LegalMoves()) > 0 {
			return
		}
		p.Turn = p.next(p.Turn)
		p.EP, p.EPPawn = NoSquare, NoSquare
	}
	g.over, g.drawn, g.method = true, true, Stuck
}

// san writes m in long algebraic notation: the piece letter (none for a
// pawn), the from square, "-" or "x", the to square, and "=Q" for a promotion.
func (g *Game) san(m Move) string { return sanAt(&g.pos, m) }

// sanAt writes m as played from p.
func sanAt(p *Position, m Move) string {
	if m.Is(FlagCastle) {
		if cell(p.Board[m.From].Color(), 6, 0) == m.To {
			return "O-O"
		}
		return "O-O-O"
	}
	var b strings.Builder
	if k := p.Board[m.From].Kind(); k != Pawn {
		b.WriteByte(k.Letter())
	}
	b.WriteString(m.From.String())
	if m.Is(FlagCapture) {
		b.WriteByte('x')
	} else {
		b.WriteByte('-')
	}
	b.WriteString(m.To.String())
	if m.Promo != NoKind {
		b.WriteByte('=')
		b.WriteByte(m.Promo.Letter())
	}
	return b.String()
}
