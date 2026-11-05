package four

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// syncGame is a Sync Move game with the four kings on their usual squares
// and the given extra pieces, like "rR@e5".
func syncGame(t *testing.T, r Rules, pieces ...string) *Game {
	t.Helper()
	r.Sync = true
	p := Position{Turn: Red, EP: NoSquare, EPPawn: NoSquare, Rules: r}
	kings := map[Color]string{Red: "h1", Blue: "a7", Green: "g14", Violet: "n8"}
	for _, s := range pieces {
		code, at, _ := strings.Cut(s, "@")
		c, _ := ParseColor(code[:1])
		if code[1] == 'K' {
			kings[c] = at
		}
	}
	for c, at := range kings {
		sq, err := ParseSquare(at)
		if err != nil {
			t.Fatal(err)
		}
		p.Board[sq] = NewPiece(c, King)
	}
	for _, s := range pieces {
		code, at, _ := strings.Cut(s, "@")
		if code[1] == 'K' {
			continue
		}
		c, _ := ParseColor(code[:1])
		sq, err := ParseSquare(at)
		if err != nil {
			t.Fatal(err)
		}
		p.Board[sq] = NewPiece(c, kindFromLetter(code[1]))
	}
	return &Game{pos: p}
}

func at(g *Game, sq string) string {
	s, _ := ParseSquare(sq)
	pc := g.pos.Board[s]
	if pc.IsEmpty() {
		return "."
	}
	return pc.Code()
}

func round(t *testing.T, g *Game, r, b, gr, v string) []Event {
	t.Helper()
	ev, err := g.PlayRound([4]string{r, b, gr, v})
	if err != nil {
		t.Fatalf("round %q %q %q %q: %v", r, b, gr, v, err)
	}
	return ev
}

func TestSyncDodge(t *testing.T) {
	g := syncGame(t, Rules{}, "rR@e5", "gN@e9")
	ev := round(t, g, "e5e9", "", "e9f11", "")
	if at(g, "e9") != "rR" || at(g, "f11") != "gN" {
		t.Fatalf("e9 %s f11 %s", at(g, "e9"), at(g, "f11"))
	}
	if !ev[0].Captured.IsEmpty() || ev[0].SAN != "Re5-e9" {
		t.Errorf("red's move: %+v", ev[0])
	}
}

func TestSyncTakesWhatStayed(t *testing.T) {
	g := syncGame(t, Rules{}, "rR@e5", "gN@e9", "gP@k13")
	ev := round(t, g, "e5e9", "", "k13k12", "")
	if at(g, "e9") != "rR" || ev[0].Captured != NewPiece(Green, Knight) || ev[0].SAN != "Re5xe9" {
		t.Fatalf("e9 %s, event %+v", at(g, "e9"), ev[0])
	}
}

func TestSyncSwap(t *testing.T) {
	g := syncGame(t, Rules{}, "rR@e5", "gR@e9")
	round(t, g, "e5e9", "", "e9e5", "")
	if at(g, "e9") != "rR" || at(g, "e5") != "gR" {
		t.Fatalf("e9 %s e5 %s", at(g, "e9"), at(g, "e5"))
	}
}

func TestSyncBump(t *testing.T) {
	g := syncGame(t, Rules{}, "rR@e5", "gR@e9", "bN@e7")
	ev := round(t, g, "e5e7", "", "e9e7", "")
	if at(g, "e7") != "." || at(g, "e5") != "." || at(g, "e9") != "." {
		t.Fatalf("e7 %s", at(g, "e7"))
	}
	for _, e := range ev {
		if !e.Bumped || !strings.HasSuffix(e.SAN, "*") {
			t.Errorf("not bumped: %+v", e)
		}
	}
}

func TestSyncKings(t *testing.T) {
	// Red and Green take each other's kings in the same round: both are out.
	g := syncGame(t, Rules{}, "rR@e5", "gK@e9", "gR@h5")
	ev := round(t, g, "e5e9", "", "h5h1", "")
	if !g.pos.Out[Red] || !g.pos.Out[Green] || g.Over() {
		t.Fatalf("out %v over %v", g.pos.Out, g.Over())
	}
	if g.Fell(Red) != KingCaptured || !strings.HasSuffix(ev[0].SAN, "#") {
		t.Errorf("fell %v, %s", g.Fell(Red), ev[0].SAN)
	}
	// Blue takes Violet's king; Blue wins.
	round(t, g, "", "a7a8", "", "n8n7")
	g2 := syncGame(t, Rules{}, "bR@b8", "vK@n8")
	round(t, g2, "h1h2", "b8n8", "g14g13", "n8m9")
	if g2.pos.Out[Violet] {
		t.Fatal("violet's king dodged")
	}
	round(t, g2, "h2h3", "n8m8", "g13g12", "m9m8")
	if !g2.pos.Out[Violet] || g2.Over() {
		t.Fatalf("violet's king bumped into the rook: out %v", g2.pos.Out)
	}
}

func TestSyncLastStanding(t *testing.T) {
	g := syncGame(t, Rules{}, "rQ@g13", "rR@a10", "rR@n10")
	round(t, g, "g13g14", "", "", "")
	round(t, g, "a10a7", "", "", "")
	if g.Over() {
		t.Fatal("over too soon")
	}
	round(t, g, "n10n8", "", "", "")
	if w, ok := g.Winner(); !ok || w != Red {
		t.Fatalf("winner %v %v", w, ok)
	}
}

func TestSyncHill(t *testing.T) {
	g := syncGame(t, Rules{Hill: true}, "rK@g6", "gK@h9")
	round(t, g, "g6g7", "", "h9h8", "")
	if !g.Over() || g.Result() != "draw" || g.Method() != KingOfTheHill {
		t.Fatalf("result %q %v", g.Result(), g.Method())
	}
	g = syncGame(t, Rules{Hill: true}, "rK@g6", "gK@h10")
	round(t, g, "g6g7", "", "h10h9", "")
	if w, ok := g.Winner(); !ok || w != Red {
		t.Fatalf("winner %v %v", w, ok)
	}
}

func TestSyncRejects(t *testing.T) {
	g := NewGame(Rules{Sync: true})
	if _, err := g.PlayRound([4]string{"e2e5", "", "", ""}); err != ErrIllegalMove {
		t.Errorf("illegal move: %v", err)
	}
	if _, err := NewGame(Rules{}).PlayRound([4]string{}); err != ErrNotSync {
		t.Errorf("turn game: %v", err)
	}
	// No en passant in sync games.
	round(t, g, "e2e4", "b7d7", "g13g11", "m8k8")
	if g.pos.EP != NoSquare {
		t.Error("en passant square set")
	}
}

func TestSyncTokens(t *testing.T) {
	g := NewGame(Rules{Sync: true})
	round(t, g, "e2e4", "b7d7", "g13g11", "m8k8")
	round(t, g, "f2f3", "", "h13h12", "m9l9")
	g.Leave(Blue, Resignation)
	round(t, g, "g1d4", "", "", "m10l10")
	toks := g.TokenList()
	want := []string{"s:e2e4,b7d7,g13g11,m8k8", "s:f2f3,-,h13h12,m9l9", "-b", "s:g1d4,-,-,m10l10"}
	if strings.Join(toks, " ") != strings.Join(want, " ") {
		t.Fatalf("tokens %v", toks)
	}
	h := NewGame(Rules{Sync: true})
	for _, tok := range toks {
		if err := h.PlayToken(tok); err != nil {
			t.Fatalf("%s: %v", tok, err)
		}
	}
	if h.pos.Board != g.pos.Board || h.pos.Out != g.pos.Out {
		t.Fatal("replay differs")
	}
}

func TestSetups(t *testing.T) {
	ok := []struct {
		l Layout
		s string
	}{
		{BackRow, "RNBKQBNR" + "PPPPPPPP" + "........"},
		{BackRow, "KRRQNNBB" + "PPPPPPPP" + "........"},
	}
	ok = append(ok, struct {
		l Layout
		s string
	}{Battleship, "PPPPPPPP" + "RRBBNN.." + "...KQ..."})
	for _, c := range ok {
		if err := CheckSetup(c.l, c.s); err != nil {
			t.Errorf("%q: %v", c.s, err)
		}
	}
	bad := []struct {
		l Layout
		s string
	}{
		{Standard, StandardSetup},
		{BackRow, "RNBQKBN" + "PPPPPPPP" + "........"},
		{BackRow, "PNBQKBNR" + "RPPPPPPP" + "........"},
		{BackRow, "RNBQKBNR" + "PPPPPPP." + ".......P"},
		{BackRow, "RNBQQBNR" + "PPPPPPPP" + "........"},
		{Battleship, "RNBQKBNR" + "PPPPPPPP" + ".......P"},
		{Battleship, "RNBQKBNR" + "PPPPPPPP" + ".......x"},
	}
	for _, c := range bad {
		if err := CheckSetup(c.l, c.s); err == nil {
			t.Errorf("%q should be refused", c.s)
		}
	}

	setups := [4]string{
		"RNBQKBNR" + "PPPPPPPP" + "........",
		"KRRQNNBB" + "PPPPPPPP" + "........",
		"PPPPPPPP" + "RRBBNN.." + "...KQ...",
		StandardSetup,
	}
	g, err := NewGameFromSetups(Rules{}, setups)
	if err != nil {
		t.Fatal(err)
	}
	p := g.Position()
	if p.Castling != 0x3|0x3<<6 {
		t.Errorf("castling %08b", p.Castling)
	}
	if p.KingSquare(Blue) != cell(Blue, 0, 0) || p.KingSquare(Green) != cell(Green, 3, 2) {
		t.Errorf("kings at %s %s", p.KingSquare(Blue), p.KingSquare(Green))
	}
	if g.Setups() != setups {
		t.Error("setups not kept")
	}
	if len(p.LegalMoves()) == 0 {
		t.Error("no moves")
	}
	std := StartPosition(Rules{})
	s, _ := SetupPosition(Rules{}, [4]string{StandardSetup, StandardSetup, StandardSetup, StandardSetup})
	if s.Board != std.Board || s.Castling != std.Castling {
		t.Error("the usual setup isn't the usual start")
	}
}

func TestBotSetup(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 20 {
		s := BotSetup(rng)
		if CheckSetup(BackRow, s) != nil || CheckSetup(Battleship, s) != nil {
			t.Fatalf("bad bot setup %q", s)
		}
	}
}
