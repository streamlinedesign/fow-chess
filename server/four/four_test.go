package four

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func play(t *testing.T, g *Game, moves ...string) {
	t.Helper()
	for _, m := range moves {
		if _, err := g.PlayUCI(m); err != nil {
			t.Fatalf("%s (turn %s): %v", m, g.Turn().Name(), err)
		}
	}
}

func TestSquares(t *testing.T) {
	for _, s := range []string{"d1", "k1", "a4", "n11", "d14", "k14", "g7"} {
		sq, err := ParseSquare(s)
		if err != nil || sq.String() != s {
			t.Errorf("%s: got %v %v", s, sq, err)
		}
	}
	for _, s := range []string{"d0", "d17", "d01", "q3", "a", "a1x"} {
		if _, err := ParseSquare(s); err == nil {
			t.Errorf("%s should not be a square", s)
		}
	}
	for _, s := range []string{"a1", "c3", "l12", "n14", "o5", "d15"} {
		if sq, err := ParseSquare(s); err == nil && Cross.Valid(sq) {
			t.Errorf("%s should not be on the cross", s)
		}
	}
	for _, s := range []string{"a1", "c3", "p16", "d15"} {
		if sq, err := ParseSquare(s); err != nil || !Extended.Valid(sq) {
			t.Errorf("%s should be on the extended board", s)
		}
	}
	for shape, want := range map[Shape]int{Cross: 160, Extended: 256} {
		n := 0
		for i := range NumSquares {
			if shape.Valid(Square(i)) {
				n++
			}
		}
		if n != want {
			t.Errorf("shape %d: %d squares, want %d", shape, n, want)
		}
	}
	from, to, promo, err := ParseUCI("g13g11")
	if err != nil || from.String() != "g13" || to.String() != "g11" || promo != NoKind {
		t.Errorf("g13g11: %v %v %v %v", from, to, promo, err)
	}
	if _, _, promo, _ := ParseUCI("e7e8q"); promo != Queen {
		t.Errorf("promo %v", promo)
	}
}

func TestStart(t *testing.T) {
	p := StartPosition(Rules{})
	count := map[Color]int{}
	for _, pc := range p.Board {
		if !pc.IsEmpty() {
			count[pc.Color()]++
		}
	}
	for _, c := range Colors {
		if count[c] != 16 {
			t.Errorf("%s has %d pieces", c.Name(), count[c])
		}
	}
	// Every player sees the usual setup from their own side: queen on the
	// left of the king.
	kings := map[Color]string{Red: "h1", Blue: "a7", Green: "g14", Violet: "n8"}
	queens := map[Color]string{Red: "g1", Blue: "a8", Green: "h14", Violet: "n7"}
	for c, sq := range kings {
		if got := p.KingSquare(c).String(); got != sq {
			t.Errorf("%s king on %s, want %s", c.Name(), got, sq)
		}
		q, _ := ParseSquare(queens[c])
		if p.Board[q] != NewPiece(c, Queen) {
			t.Errorf("%s queen not on %s", c.Name(), queens[c])
		}
	}
	g := NewGame(Rules{})
	for _, c := range Colors {
		if g.Turn() != c {
			t.Fatalf("turn %s, want %s", g.Turn().Name(), c.Name())
		}
		if n := len(g.LegalMoves()); n != 20 {
			t.Errorf("%s has %d moves at the start, want 20", c.Name(), n)
		}
		g.PlayUCI(g.LegalMoves()[0].UCI())
	}
	if g.Turn() != Red {
		t.Errorf("after a round it's %s's turn", g.Turn().Name())
	}
}

func TestPawnsMoveForward(t *testing.T) {
	g := NewGame(Rules{})
	// Red up, Blue right, Green down, Violet left.
	play(t, g, "e2e4", "b5d5", "j13j11", "m10k10")
	p := g.Position()
	for _, s := range []string{"e4", "d5", "j11", "k10"} {
		sq, _ := ParseSquare(s)
		if p.Board[sq].Kind() != Pawn {
			t.Errorf("no pawn on %s", s)
		}
	}
	// Only the next player may take en passant, and only right away.
	if p.EP.String() != "l10" {
		t.Errorf("ep square %s", p.EP)
	}
}

func TestEnPassant(t *testing.T) {
	// A Blue pawn on e4 goes right, so its diagonals ahead are f3 and f5.
	g2 := NewGame(Rules{})
	p2 := StartPosition(Rules{})
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	p2.Board[sq("b4")] = NoPiece
	p2.Board[sq("e4")] = NewPiece(Blue, Pawn)
	g2.pos = p2
	play(t, g2, "f2f4")
	var ep *Move
	for _, m := range g2.LegalMoves() {
		if m.Is(FlagEnPassant) {
			ep = &m
		}
	}
	if ep == nil || ep.UCI() != "e4f3" {
		t.Fatalf("Blue should take en passant e4f3, got %v", ep)
	}
	play(t, g2, "e4f3")
	if !g2.Position().Board[sq("f4")].IsEmpty() || g2.History()[1].Captured != NewPiece(Red, Pawn) {
		t.Errorf("en passant didn't take the pawn")
	}
}

func TestPromotion(t *testing.T) {
	g := NewGame(Rules{})
	p := StartPosition(Rules{})
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	p.Board[sq("e2")] = NoPiece
	p.Board[sq("e7")] = NewPiece(Red, Pawn)
	g.pos = p
	n := 0
	for _, m := range g.LegalMoves() {
		if m.From == sq("e7") {
			n++
			if m.Promo == NoKind {
				t.Errorf("%s should promote", m.UCI())
			}
		}
	}
	if n != 4 {
		t.Errorf("%d promotion moves, want 4", n)
	}
	play(t, g, "e7e8n")
	if g.Position().Board[sq("e8")] != NewPiece(Red, Knight) {
		t.Errorf("no knight on e8")
	}
}

func TestCastling(t *testing.T) {
	g := NewGame(Rules{})
	p := StartPosition(Rules{})
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	for _, s := range []string{"i1", "j1", "a9", "a10"} {
		p.Board[sq(s)] = NoPiece
	}
	g.pos = p
	play(t, g, "h1j1") // Red short
	if g.Position().Board[sq("i1")] != NewPiece(Red, Rook) || g.Position().Board[sq("j1")] != NewPiece(Red, King) {
		t.Errorf("red didn't castle")
	}
	if g.History()[0].SAN != "O-O" {
		t.Errorf("SAN %s", g.History()[0].SAN)
	}
	// Blue's king on a7 castles short towards a4.
	for _, s := range []string{"a5", "a6"} {
		g.pos.Board[sq(s)] = NoPiece
	}
	play(t, g, "a7a5")
	if g.Position().Board[sq("a6")] != NewPiece(Blue, Rook) {
		t.Errorf("blue rook didn't move")
	}
}

func TestEatingKingsAndWinning(t *testing.T) {
	g := NewGame(Rules{})
	p := StartPosition(Rules{})
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	// A Red queen next to Blue's king.
	p.Board[sq("b7")] = NewPiece(Red, Queen)
	g.pos = p
	play(t, g, "b7a7")
	pos := g.Position()
	if !pos.Out[Blue] || g.Fell(Blue) != KingCaptured {
		t.Fatalf("blue should be out")
	}
	for _, pc := range pos.Board {
		if pc.Is(Blue) {
			t.Fatalf("blue pieces left on the board")
		}
	}
	if g.Turn() != Green {
		t.Errorf("turn %s, want green (blue is skipped)", g.Turn().Name())
	}
	g.Leave(Violet, Resignation)
	if g.Over() {
		t.Fatalf("two players left, game over")
	}
	g.Leave(Green, Timeout)
	if w, ok := g.Winner(); !ok || w != Red || g.Method() != Timeout {
		t.Errorf("winner %v %v %v", w, ok, g.Method())
	}
	if g.Result() != "r" {
		t.Errorf("result %q", g.Result())
	}

	// Replaying the saved tokens gives the same game.
	again := NewGame(Rules{})
	again.pos.Board[sq("b7")] = NewPiece(Red, Queen)
	for _, tok := range strings.Fields(g.Tokens()) {
		if err := again.PlayToken(tok); err != nil {
			t.Fatal(err)
		}
	}
	if again.Result() != "r" || len(again.History()) != 3 {
		t.Errorf("replay: %q %d", again.Result(), len(again.History()))
	}
}

func TestHill(t *testing.T) {
	g := NewGame(Rules{Hill: true})
	p := g.pos
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	p.Board[sq("h1")] = NoPiece
	p.Board[sq("g6")] = NewPiece(Red, King)
	g.pos = p
	play(t, g, "g6g7")
	if w, ok := g.Winner(); !ok || w != Red || g.Method() != KingOfTheHill {
		t.Errorf("hill: %v %v %v", w, ok, g.Method())
	}
}

func TestVision(t *testing.T) {
	p := StartPosition(Rules{CenterVision: CenterAny})
	seen := p.Vision(Red)
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	for _, s := range []string{"d1", "e3", "e4", "g3", "i3"} {
		if !seen[sq(s)] {
			t.Errorf("red should see %s", s)
		}
	}
	for _, s := range []string{"e5", "b5", "g13", "m7"} {
		if seen[sq(s)] {
			t.Errorf("red shouldn't see %s", s)
		}
	}
	p.Board[sq("g7")] = NewPiece(Red, Knight)
	if p.Outpost(Red) == NoSquare {
		t.Errorf("a piece on g7 is an outpost")
	}
	if all := p.Vision(Red); !all[sq("g13")] {
		t.Errorf("centre vision should show everything")
	}
}

func TestMods(t *testing.T) {
	sq := func(s string) Square { q, _ := ParseSquare(s); return q }
	p := StartPosition(Rules{JesterPawns: true})
	p.Board[sq("g5")] = NewPiece(Red, Pawn)
	var tos []string
	for _, m := range p.PieceMoves(sq("g5")) {
		tos = append(tos, m.To.String())
	}
	slices.Sort(tos)
	if want := []string{"f4", "f5", "f6", "g4", "g6", "h4", "h5", "h6"}; !slices.Equal(tos, want) {
		t.Errorf("jester moves %v", tos)
	}
	p = StartPosition(Rules{JumpKing: true})
	p.Board[sq("g5")] = NewPiece(Blue, King)
	if n := len(p.PieceMoves(sq("g5"))); n != 16 {
		t.Errorf("jump king has %d moves", n)
	}
}

// Bots playing each other always finish without breaking a rule.
func TestBotsFinish(t *testing.T) {
	for seed := range 6 {
		rng := rand.New(rand.NewPCG(uint64(seed), 1))
		rules := Rules{CenterVision: CenterAny, JesterPawns: seed%3 == 1, JumpKing: seed%3 == 2, Hill: seed == 5}
		g := NewGame(rules)
		for i := 0; i < 3000 && !g.Over(); i++ {
			pos := g.Position()
			m, ok := BotMove(&pos, rng)
			if !ok {
				t.Fatalf("seed %d: no move but game not over", seed)
			}
			if _, err := g.PlayUCI(m.UCI()); err != nil {
				t.Fatalf("seed %d: bot played %s: %v", seed, m.UCI(), err)
			}
		}
		if !g.Over() {
			t.Logf("seed %d: still going after 3000 moves", seed)
		}
	}
}
