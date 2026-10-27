package engine

import "errors"

type Outcome uint8

const (
	Ongoing Outcome = iota
	WhiteWins
	BlackWins
	Drawn
)

// String returns the PGN result: "1-0", "0-1", "1/2-1/2", or "*".
func (o Outcome) String() string {
	return [...]string{"*", "1-0", "0-1", "1/2-1/2"}[o]
}

func winFor(c Color) Outcome {
	if c == White {
		return WhiteWins
	}
	return BlackWins
}

type Method uint8

const (
	NoMethod Method = iota
	Checkmate
	Resignation
	Stalemate
	Agreement
	Repetition
	FiftyMoves
	InsufficientMaterial
	KingCaptured
)

func (m Method) String() string {
	return [...]string{"", "checkmate", "resignation", "stalemate", "agreement", "repetition", "fifty-move rule", "insufficient material", "capturing the king"}[m]
}

// PlayedMove is one entry in a game's history.
type PlayedMove struct {
	Move Move
	SAN  string
}

// Game tracks a position plus the history the draw rules need.
type Game struct {
	pos     Position
	history []PlayedMove
	seen    map[string]int // repetition counts by position key
	outcome Outcome
	method  Method
}

var ErrGameOver = errors.New("game is over")
var ErrIllegalMove = errors.New("illegal move")

func NewGame() *Game {
	g, _ := NewGameFromFEN(StartFEN)
	return g
}

// Rules are the settings a game is played under.
type Rules struct {
	// KingCapture plays without check: win by capturing the king.
	KingCapture bool
}

func NewGameWithRules(r Rules) *Game {
	g, _ := NewGameFromFEN(StartFEN, r)
	return g
}

func NewGameFromFEN(fen string, rules ...Rules) (*Game, error) {
	pos, err := ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	if len(rules) > 0 {
		pos.KingCapture = rules[0].KingCapture
	}
	g := &Game{pos: pos, seen: map[string]int{}}
	g.seen[pos.repetitionKey()] = 1
	g.evaluate()
	return g, nil
}

func (g *Game) Position() Position    { return g.pos }
func (g *Game) Turn() Color           { return g.pos.Turn }
func (g *Game) History() []PlayedMove { return g.history }
func (g *Game) Outcome() Outcome      { return g.outcome }
func (g *Game) Method() Method        { return g.method }
func (g *Game) Over() bool            { return g.outcome != Ongoing }
func (g *Game) LegalMoves() []Move    { return g.pos.LegalMoves() }

// PlayUCI plays a move given as "e2e4" / "e7e8q" if it's legal.
func (g *Game) PlayUCI(uci string) (PlayedMove, error) {
	if g.Over() {
		return PlayedMove{}, ErrGameOver
	}
	m, ok := g.pos.FindMove(uci)
	if !ok {
		return PlayedMove{}, ErrIllegalMove
	}
	return g.play(m), nil
}

func (g *Game) play(m Move) PlayedMove {
	san := g.pos.SAN(m)
	g.pos = g.pos.Apply(m)
	pm := PlayedMove{Move: m, SAN: san}
	g.history = append(g.history, pm)
	g.seen[g.pos.repetitionKey()]++
	g.evaluate()
	return pm
}

// Resign ends the game in the opponent's favor.
func (g *Game) Resign(c Color) {
	if !g.Over() {
		g.outcome, g.method = winFor(c.Other()), Resignation
	}
}

// AgreeDraw ends the game as a draw by agreement.
func (g *Game) AgreeDraw() {
	if !g.Over() {
		g.outcome, g.method = Drawn, Agreement
	}
}

// evaluate applies the automatic end-of-game rules. Threefold repetition and the
// fifty-move rule end the game on their own, as on most online sites.
func (g *Game) evaluate() {
	p := &g.pos
	switch {
	case p.KingSquare(p.Turn) == NoSquare:
		g.outcome, g.method = winFor(p.Turn.Other()), KingCaptured
	case !p.HasLegalMove():
		if p.InCheck() {
			g.outcome, g.method = winFor(p.Turn.Other()), Checkmate
		} else {
			g.outcome, g.method = Drawn, Stalemate
		}
	case p.insufficientMaterial():
		g.outcome, g.method = Drawn, InsufficientMaterial
	case g.seen[p.repetitionKey()] >= 3:
		g.outcome, g.method = Drawn, Repetition
	case p.Halfmove >= 100:
		g.outcome, g.method = Drawn, FiftyMoves
	}
}

// insufficientMaterial covers the positions where no sequence of moves can mate:
// king vs king, king and one minor piece vs king, and kings with bishops that all
// stand on the same square color.
func (p *Position) insufficientMaterial() bool {
	minors, knights := 0, 0
	bishopColors := [2]bool{}
	for sq, pc := range p.Board {
		switch pc.Kind() {
		case NoKind, King:
		case Knight:
			minors++
			knights++
		case Bishop:
			minors++
			s := Square(sq)
			bishopColors[(s.File()+s.Rank())%2] = true
		default:
			return false
		}
	}
	if minors <= 1 {
		return true
	}
	return knights == 0 && !(bishopColors[0] && bishopColors[1])
}
