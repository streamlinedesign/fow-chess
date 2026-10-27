package main

import (
	"strings"
	"testing"

	"chess/engine"
)

func TestViewOnlyGivesMovesToPlayerOnTurn(t *testing.T) {
	g := engine.NewGame()
	if v := viewFor(g, white, true, false); len(v.Legal) != 20 {
		t.Errorf("white to move: got %d legal moves, want 20", len(v.Legal))
	}
	for name, viewer := range map[string]int{"black": black, "spectator": -1} {
		if v := viewFor(g, viewer, true, false); len(v.Legal) != 0 {
			t.Errorf("%s got %d legal moves on white's turn", name, len(v.Legal))
		}
	}
	if v := viewFor(g, white, false, false); len(v.Legal) != 0 {
		t.Error("legal moves sent while the match isn't playable")
	}
}

func TestViewAfterMoves(t *testing.T) {
	g := engine.NewGame()
	for _, uci := range []string{"e2e4", "f7f6", "d1h5"} {
		if _, err := g.PlayUCI(uci); err != nil {
			t.Fatal(err)
		}
	}
	v := viewFor(g, black, true, false)
	// Ranks 1 to 8: pawn on e4, queen on h5, pawn on f6.
	want := "RNB.KBNR" + "PPPP.PPP" + "........" + "....P..." + ".......Q" + ".....p.." + "ppppp.pp" + "rnbqkbnr"
	if v.Board != want {
		t.Errorf("board\n got %s\nwant %s", v.Board, want)
	}
	if v.Ply != 3 || v.Turn != "b" || v.Check != "e8" || v.Last.From != "d1" || v.Last.To != "h5" {
		t.Errorf("unexpected view %+v", v)
	}
	if got := v.History[2]; got != "Qh5+" {
		t.Errorf("SAN %s", got)
	}
	for _, m := range v.Legal {
		if m != "g7g6" && m != "e8f7" {
			t.Errorf("unexpected legal move out of check: %s", m)
		}
	}
}

func TestFogViews(t *testing.T) {
	g := engine.NewGame()
	for _, uci := range []string{"e2e4", "e7e5", "g1f3", "b8c6"} {
		if _, err := g.PlayUCI(uci); err != nil {
			t.Fatal(err)
		}
	}

	w := viewFor(g, white, true, true)
	black := 0
	for _, c := range w.Board {
		if strings.ContainsRune("rnbqkp", c) {
			black++
		}
	}
	if black != 1 {
		t.Errorf("white should see exactly one black piece, sees %d: %s", black, w.Board)
	}
	if w.Board[36] != 'p' { // e5 pawn, seen because it blocks e4
		t.Errorf("white should see the e5 pawn blocking its own: %q", w.Board[36])
	}
	if w.Last != nil {
		t.Errorf("white got a hint about black's last move: %+v", w.Last)
	}
	if got := strings.Join(w.History, " "); got != "e4 ? Nf3 ?" {
		t.Errorf("white history %q", got)
	}
	if len(w.Legal) == 0 {
		t.Error("white to move has no legal moves")
	}

	b := viewFor(g, black, true, true)
	if b.Last == nil || b.Last.To != "c6" {
		t.Errorf("black should see its own last move: %+v", b.Last)
	}
	if got := strings.Join(b.History, " "); got != "? e5 ? Nc6" {
		t.Errorf("black history %q", got)
	}

	s := viewFor(g, -1, true, true)
	visible := 0
	for _, c := range s.Board {
		if c != '?' {
			visible++
		}
	}
	if visible != 4 || s.Board[21] != 'N' || s.Board[42] != 'n' { // f3 and c6
		t.Errorf("spectator should see just the last two moves: %s", s.Board)
	}
	if got := strings.Join(s.History, " "); got != "? ? Nf3 Nc6" {
		t.Errorf("spectator history %q", got)
	}

	// Check marks on your own moves would reveal the enemy king.
	g2 := engine.NewGame()
	for _, uci := range []string{"e2e4", "f7f6", "d1h5"} {
		g2.PlayUCI(uci)
	}
	if v := viewFor(g2, white, true, true); v.History[2] != "Qh5" || v.Check != "" {
		t.Errorf("white learned about check: %v %q", v.History, v.Check)
	}
	if v := viewFor(g2, black, true, true); v.Check != "e8" {
		t.Errorf("black should know it is in check, got %q", v.Check)
	}

	// After the game everything is revealed.
	g.Resign(white)
	if v := viewFor(g, white, true, true); strings.Contains(v.Board, "?") || v.History[1] != "e5" {
		t.Errorf("finished game still fogged: %s %v", v.Board, v.History)
	}
}

func TestCapturedPieces(t *testing.T) {
	g := engine.NewGame()
	// White takes a pawn, then one en passant, then a knight.
	for _, uci := range []string{"e2e4", "d7d5", "e4d5", "g8f6", "c2c4", "c7c5", "d5c6", "f6e4", "d2d3", "a7a6", "d3e4"} {
		if _, err := g.PlayUCI(uci); err != nil {
			t.Fatal(uci, err)
		}
	}
	for _, viewer := range []int{white, black, -1} {
		v := viewFor(g, viewer, true, true)
		if v.Captured["w"] != "npp" || v.Captured["b"] != "" {
			t.Errorf("viewer %d: captured %v", viewer, v.Captured)
		}
	}
}
