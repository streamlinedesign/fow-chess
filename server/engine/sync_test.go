package engine

import (
	"slices"
	"testing"
)

func syncGame(t *testing.T, fen string, r Rules) *Game {
	t.Helper()
	r.KingCapture, r.Sync = true, true
	g, err := NewGameFromFEN(fen, r)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func round(t *testing.T, g *Game, w, b string) [2]PlayedMove {
	t.Helper()
	pm, err := g.PlayRound([2]string{w, b})
	if err != nil {
		t.Fatalf("round %q %q: %v", w, b, err)
	}
	return pm
}

func at(g *Game, sq string) byte {
	s, _ := ParseSquare(sq)
	pc := g.Position().Board[s]
	if pc.IsEmpty() {
		return '.'
	}
	return pc.Letter()
}

func TestSyncDodgeSwapBump(t *testing.T) {
	// White rook a1, black knight a5, kings far apart.
	g := syncGame(t, "4k3/8/8/n7/8/8/8/R3K3 w - - 0 1", Rules{})
	pm := round(t, g, "a1a5", "a5c6")
	if at(g, "a5") != 'R' || at(g, "c6") != 'n' || !pm[0].Captured.IsEmpty() || pm[0].SAN != "Ra5" {
		t.Fatalf("dodge: a5 %c c6 %c %+v", at(g, "a5"), at(g, "c6"), pm[0])
	}
	g = syncGame(t, "4k3/8/8/r7/8/8/8/R3K3 w - - 0 1", Rules{})
	round(t, g, "a1a5", "a5a1")
	if at(g, "a5") != 'R' || at(g, "a1") != 'r' {
		t.Fatalf("swap: a5 %c a1 %c", at(g, "a5"), at(g, "a1"))
	}
	g = syncGame(t, "4k3/8/8/r7/8/8/8/R3K3 w - - 0 1", Rules{})
	pm = round(t, g, "a1a3", "a5a3")
	if at(g, "a3") != '.' || pm[0].SAN != "Ra3*" || pm[1].Captured.Letter() != 'R' {
		t.Fatalf("bump: a3 %c %+v", at(g, "a3"), pm)
	}
	g = syncGame(t, "4k3/8/8/n7/8/8/8/R3K3 w - - 0 1", Rules{})
	pm = round(t, g, "a1a5", "e8d8")
	if at(g, "a5") != 'R' || pm[0].Captured.Letter() != 'n' || pm[0].SAN != "Rxa5" {
		t.Fatalf("take: %+v", pm[0])
	}
	if g.Position().Turn != White || g.Position().EP != NoSquare {
		t.Error("turn or en passant")
	}
}

func TestSyncEnds(t *testing.T) {
	// Each rook takes the other king in the same round: a draw.
	g := syncGame(t, "4k2R/8/8/8/8/8/8/r3K3 w - - 0 1", Rules{})
	round(t, g, "h8e8", "a1e1")
	if g.Outcome() != Drawn || g.Method() != KingCaptured {
		t.Fatalf("both kings: %v %v", g.Outcome(), g.Method())
	}
	g = syncGame(t, "4k2R/8/8/8/8/8/8/r3K3 w - - 0 1", Rules{})
	round(t, g, "h8e8", "a1b1")
	if g.Outcome() != WhiteWins {
		t.Fatalf("white eats: %v", g.Outcome())
	}
	// The black king steps out of the way.
	g = syncGame(t, "4k2R/8/8/8/8/8/8/r3K3 w - - 0 1", Rules{})
	round(t, g, "h8e8", "e8d7")
	if g.Over() || at(g, "e8") != 'R' {
		t.Fatal("king should dodge")
	}
	g = syncGame(t, "8/8/3k4/8/8/4K3/8/8 w - - 0 1", Rules{Hill: true})
	round(t, g, "e3e4", "d6d5")
	if g.Outcome() != Drawn || g.Method() != KingOfTheHill {
		t.Fatalf("hill: %v %v", g.Outcome(), g.Method())
	}
	g = syncGame(t, StartFEN, Rules{})
	g.Flag(Black)
	if g.Outcome() != WhiteWins || g.Method() != Timeout {
		t.Fatalf("flag: %v", g.Outcome())
	}
}

func TestSyncTokens(t *testing.T) {
	g := syncGame(t, StartFEN, Rules{})
	if _, err := g.PlayUCI("e2e4"); err == nil {
		t.Error("a single move in a sync game")
	}
	round(t, g, "e2e4", "e7e5")
	round(t, g, "g1f3", "b8c6")
	round(t, g, "f1c4", "")
	toks := g.Tokens()
	want := []string{"s:e2e4,e7e5", "s:g1f3,b8c6", "s:f1c4,-"}
	if !slices.Equal(toks, want) {
		t.Fatalf("tokens %v", toks)
	}
	if !g.History()[5].IsPass() || len(g.History()) != 6 {
		t.Fatal("pass entry")
	}
	h := syncGame(t, StartFEN, Rules{})
	for _, tok := range toks {
		if err := h.PlayToken(tok); err != nil {
			t.Fatal(err)
		}
	}
	if h.Position().Board != g.Position().Board {
		t.Fatal("replay differs")
	}
	if _, err := g.PlayRound([2]string{"e4e6", ""}); err != ErrIllegalMove {
		t.Errorf("illegal: %v", err)
	}
	if _, err := NewGame().PlayRound([2]string{}); err != ErrNotSync {
		t.Errorf("turns: %v", err)
	}
	// Castling moves the rook too.
	g = syncGame(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", Rules{})
	round(t, g, "e1g1", "e8c8")
	if at(g, "f1") != 'R' || at(g, "d8") != 'r' || g.Position().Castling != 0 {
		t.Fatalf("castling: f1 %c d8 %c %b", at(g, "f1"), at(g, "d8"), g.Position().Castling)
	}
}
