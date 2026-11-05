package four

import (
	"math/rand/v2"
	"testing"
)

func sqs(t *testing.T, names ...string) []Square {
	t.Helper()
	out := make([]Square, len(names))
	for i, n := range names {
		sq, err := ParseSquare(n)
		if err != nil {
			t.Fatal(err)
		}
		out[i] = sq
	}
	return out
}

func TestExtendedStart(t *testing.T) {
	r := Rules{Shape: Extended}
	p := StartPosition(r)
	kings := map[Color]string{Red: "i1", Blue: "a8", Green: "h16", Violet: "p9"}
	queens := map[Color]string{Red: "h1", Blue: "a9", Green: "i16", Violet: "p8"}
	for c, s := range kings {
		if got := p.KingSquare(c).String(); got != s {
			t.Errorf("%s king on %s, want %s", c.Name(), got, s)
		}
		if q := sqs(t, queens[c])[0]; p.Board[q] != NewPiece(c, Queen) {
			t.Errorf("%s queen not on %s", c.Name(), queens[c])
		}
	}
	// Red's pawns start on the second row and step two to the fourth.
	g := NewGame(r)
	play(t, g, "e2e4", "b5d5", "l15l13", "o12m12")
	for _, c := range Colors {
		if g.Position().Board[p.Shape.Cell(c, 0, 0)] != NewPiece(c, Rook) {
			t.Errorf("%s rook moved", c.Name())
		}
	}
}

// The corners are neutral squares anyone can cross.
func TestExtendedCorners(t *testing.T) {
	var p Position
	p.Rules = Rules{Shape: Extended}
	p.EP, p.EPPawn = NoSquare, NoSquare
	rook := sqs(t, "e1")[0]
	p.Board[rook] = NewPiece(Red, Rook)
	reach := map[string]bool{}
	for _, m := range p.PieceMoves(rook) {
		reach[m.To.String()] = true
	}
	for _, s := range []string{"a1", "d1", "e16", "p1"} {
		if !reach[s] {
			t.Errorf("rook on e1 should reach %s", s)
		}
	}
	// The same rook on the cross stops at the cut corner.
	p.Shape = Cross
	for _, m := range p.PieceMoves(rook) {
		if m.To.X() < 3 {
			t.Errorf("rook on the cross reached %s", m.To)
		}
	}
}

func TestExtendedPromotion(t *testing.T) {
	p := StartPosition(Rules{Shape: Extended})
	at := sqs(t, "e8", "e15", "e16")
	p.Board[at[2]] = NoPiece
	p.Board[at[0]] = NewPiece(Red, Pawn)
	p.Board[at[1]] = NewPiece(Red, Pawn)
	for _, m := range p.PieceMoves(at[0]) {
		if m.Promo != NoKind {
			t.Errorf("%s promoted halfway", m.UCI())
		}
	}
	n := 0
	for _, m := range p.PieceMoves(at[1]) {
		if m.To == at[2] && m.Promo != NoKind {
			n++
		}
	}
	if n != 4 {
		t.Errorf("%d promotions on the far edge, want 4", n)
	}
}

func TestExtendedOutposts(t *testing.T) {
	want := map[Color][2]string{Red: {"h9", "i9"}, Blue: {"i8", "i9"}, Green: {"h8", "i8"}, Violet: {"h8", "h9"}}
	for c, names := range want {
		got := Extended.Outposts(c)
		if len(got) != 2 || got[0].String() != names[0] || got[1].String() != names[1] {
			t.Errorf("%s outposts %v, want %v", c.Name(), got, names)
		}
	}
	p := StartPosition(Rules{Shape: Extended, CenterVision: CenterAny})
	p.Board[sqs(t, "h8")[0]] = NewPiece(Red, Knight)
	if p.Outpost(Red) != NoSquare {
		t.Errorf("h8 is on red's own side")
	}
	p.Board[sqs(t, "i9")[0]] = NewPiece(Red, Knight)
	if p.Outpost(Red) == NoSquare || !p.Vision(Red)[sqs(t, "h16")[0]] {
		t.Errorf("i9 should show red everything")
	}
}

func TestExtendedNoHill(t *testing.T) {
	r := Rules{Shape: Extended, Hill: true}
	if r.HillOn() {
		t.Fatal("no hill on the extended board")
	}
	g := NewGame(r)
	p := g.Position()
	p.Board[p.KingSquare(Red)] = NoPiece
	p.Board[sqs(t, "h7")[0]] = NewPiece(Red, King)
	g.pos = p
	play(t, g, "h7h8")
	if g.Over() {
		t.Errorf("game ended by %s", g.Method())
	}
}

func TestExtendedSetups(t *testing.T) {
	std := Extended.StandardSetup()
	if len(std) != 32 {
		t.Fatalf("setup length %d", len(std))
	}
	deep := "................................"
	deep = "RNBQKBNR" + deep[8:24] + "PPPPPPPP"
	for _, s := range []string{std, deep} {
		if err := CheckSetup(Extended, Battleship, s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
	if CheckSetup(Cross, Battleship, deep[:24]) == nil {
		t.Errorf("cross setup without pawns accepted")
	}
	if CheckSetup(Extended, Battleship, Cross.StandardSetup()) == nil {
		t.Errorf("24-square setup accepted on the extended board")
	}
	rng := rand.New(rand.NewPCG(3, 4))
	b := BotSetup(Extended, rng)
	if err := CheckSetup(Extended, BackRow, b); err != nil {
		t.Errorf("bot setup %q: %v", b, err)
	}
	p, err := SetupPosition(Rules{Shape: Extended}, [4]string{deep, std, std, std})
	if err != nil {
		t.Fatal(err)
	}
	if p.Board[sqs(t, "e4")[0]] != NewPiece(Red, Pawn) || p.KingSquare(Red).String() != "i1" {
		t.Errorf("red's setup not placed")
	}
}

func TestExtendedBotsFinish(t *testing.T) {
	for seed := range 3 {
		rng := rand.New(rand.NewPCG(uint64(seed), 7))
		g := NewGame(Rules{Shape: Extended, CenterVision: CenterAny, Sync: seed == 2})
		for i := 0; i < 3000 && !g.Over(); i++ {
			pos := g.Position()
			if pos.Sync {
				var moves [4]string
				for _, c := range Colors {
					q := pos
					q.Turn = c
					if m, ok := BotMove(&q, rng); ok && !pos.Out[c] {
						moves[c] = m.UCI()
					}
				}
				if _, err := g.PlayRound(moves); err != nil {
					t.Fatal(err)
				}
				continue
			}
			m, ok := BotMove(&pos, rng)
			if !ok {
				t.Fatal("no bot move")
			}
			if _, err := g.PlayUCI(m.UCI()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// On the extended board a pawn promotes on any edge but its own side.
func TestExtendedPromotionEdges(t *testing.T) {
	var p Position
	p.Rules = Rules{Shape: Extended, JesterPawns: true}
	p.EP, p.EPPawn = NoSquare, NoSquare
	promos := func(from string) map[string]bool {
		out := map[string]bool{}
		p.Board = [NumSquares]Piece{}
		f := sqs(t, from)[0]
		p.Board[f] = NewPiece(Red, Pawn)
		for _, m := range p.PieceMoves(f) {
			if m.Promo != NoKind {
				out[m.To.String()] = true
			}
		}
		return out
	}
	if got := promos("b9"); !got["a9"] || !got["a10"] || !got["a8"] || got["b10"] {
		t.Errorf("from b9: %v", got)
	}
	if got := promos("o12"); !got["p12"] || !got["p13"] {
		t.Errorf("from o12: %v", got)
	}
	if got := promos("b2"); len(got) != 0 {
		t.Errorf("a jester pawn on its own rows promoted: %v", got)
	}
	if got := promos("b4"); !got["a5"] || got["a4"] || got["a3"] {
		t.Errorf("from b4: %v", got)
	}
	// Blue's pawns go right; its far edge is the p-file and its sides the
	// top and bottom edges past its own rows.
	p.Board = [NumSquares]Piece{}
	f := sqs(t, "e2")[0]
	p.Board[f] = NewPiece(Blue, Pawn)
	got := map[string]bool{}
	for _, m := range p.PieceMoves(f) {
		if m.Promo != NoKind {
			got[m.To.String()] = true
		}
	}
	if !got["e1"] || !got["f1"] || got["d1"] {
		t.Errorf("blue from e2: %v", got)
	}
}
