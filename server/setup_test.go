package main

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"chess/engine"
)

// place moves the piece on from to to in a setup board.
func place(board, from, to string) string {
	sq := func(s string) int { return int(s[0]-'a') + int(s[1]-'1')*8 }
	b := []byte(board)
	b[sq(to)], b[sq(from)] = b[sq(from)], '.'
	return string(b)
}

func TestCheckSetup(t *testing.T) {
	back, ship := setupModes["backRow"], setupModes["battleship"]
	for _, c := range []int{white, black} {
		if err := checkSetup(back, c, ownSetup(c)); err != nil {
			t.Errorf("the usual position as a back row setup for %d: %v", c, err)
		}
		if err := checkSetup(ship, c, ownSetup(c)); err != nil {
			t.Errorf("the usual position as a battleship setup for %d: %v", c, err)
		}
	}
	tests := []struct {
		name  string
		mode  setupMode
		color int
		board string
		ok    bool
	}{
		{"back row king and queen swapped", back, white, swap(ownSetup(white), "d1", "e1"), true},
		{"back row moves a pawn", back, white, place(ownSetup(white), "e2", "e3"), false},
		{"back row piece off the back rank", back, black, place(ownSetup(black), "b8", "c6"), false},
		{"battleship pawn on the fourth row", ship, white, place(ownSetup(white), "e2", "e4"), true},
		{"battleship knight on black's fourth row", ship, black, place(ownSetup(black), "g8", "g5"), true},
		{"battleship past the fourth row", ship, white, place(ownSetup(white), "e2", "e5"), false},
		{"a piece left out", ship, white, strings.Replace(ownSetup(white), "R", ".", 1), false},
		{"the opponent's piece", ship, white, ownSetup(white)[:56] + "r.......", false},
		{"usual position has no setup", setupModes[""], white, ownSetup(white), false},
	}
	for _, tc := range tests {
		if err := checkSetup(tc.mode, tc.color, tc.board); (err == nil) != tc.ok {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}

func swap(board, a, b string) string {
	sq := func(s string) int { return int(s[0]-'a') + int(s[1]-'1')*8 }
	x := []byte(board)
	x[sq(a)], x[sq(b)] = x[sq(b)], x[sq(a)]
	return string(x)
}

func TestSetupFEN(t *testing.T) {
	if got := setupFEN([2]string{ownSetup(white), ownSetup(black)}); got != engine.StartFEN {
		t.Errorf("usual setups make %q", got)
	}
	// A king off its square loses both castles; a rook off its square loses that side.
	got := setupFEN([2]string{swap(ownSetup(white), "e1", "d1"), place(ownSetup(black), "h8", "h6")})
	if !strings.HasSuffix(got, " w q - 0 1") {
		t.Errorf("castling in %q", got)
	}
}

func TestSniperWin(t *testing.T) {
	cm, kc := engine.Checkmate.String(), engine.KingCaptured.String()
	for _, tc := range []struct {
		result, reason string
		plies          int
		want           bool
	}{
		{"1-0", kc, 1, true},
		{"0-1", kc, 2, true},
		{"0-1", cm, 0, true},
		{"1-0", kc, 3, false},
		{"0-1", kc, 1, false},
		{"1-0", "resignation", 1, false},
		{"1/2-1/2", "stalemate", 0, false},
	} {
		if got := sniperWin(tc.result, tc.reason, tc.plies); got != tc.want {
			t.Errorf("%s by %s after %d: %v", tc.result, tc.reason, tc.plies, got)
		}
	}
}

func TestSetupMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	store := openTestStore(t, path)
	ann, _ := store.CreateUser("ann", []byte("x"))
	ben, _ := store.CreateUser("ben", []byte("x"))
	hub := NewHub(store)
	pa, pb := userPlayer(ann, "a"), userPlayer(ben, "b")

	settings := Settings{Setup: "battleship", Fog: true, KingCapture: true, Time: TimeControl{Base: 300}}
	m := hub.Create(pa, settings)
	cw, cb := newClient(pa.key), newClient(pb.key)
	m.join(cw, pa)
	m.join(cb, pb)
	if s := lastState(t, cw); s["status"] != "setup" || s["clock"].(map[string]any)["running"] != nil {
		t.Fatalf("before setting up: status %v clock %v", s["status"], s["clock"])
	}

	// Nothing moves until both have set up.
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	if m.game.History() != nil {
		t.Fatal("a move was played during setup")
	}

	// White aims a rook up the open e-file; Black leaves it open.
	ws := place(place(ownSetup(white), "e2", "h3"), "h1", "e3")
	bs := place(ownSetup(black), "e7", "a5")
	m.handle(cw, inMsg{Type: "setup", Board: place(ws, "e1", "e8")}) // not White's rows: ignored
	if s := lastState(t, cb); s["setup"].(map[string]any)["ready"].(map[string]any)["w"] != false {
		t.Error("a bad setup counted as ready")
	}
	m.handle(cw, inMsg{Type: "setup", Board: ws})
	s := setupOf(t, lastState(t, cb))
	if s["ready"].(map[string]any)["w"] != true || s["theirs"] != nil {
		t.Errorf("black's view of white's setup: %v", s)
	}
	if setupOf(t, lastState(t, cw))["mine"] != ws {
		t.Error("white's own setup isn't sent back")
	}

	m.handle(cb, inMsg{Type: "setup", Board: bs})
	st := lastState(t, cw)
	if st["status"] != "playing" || st["clock"].(map[string]any)["running"] != "w" || st["setup"] != nil {
		t.Fatalf("after both set up: status %v clock %v", st["status"], st["clock"])
	}

	// The king started in the rook's sights: taking it on the first move is a Sniper Win.
	m.handle(cw, inMsg{Type: "move", UCI: "e3e8"})
	st = lastState(t, cw)
	if st["result"] != "1-0" || st["sniper"] != true {
		t.Fatalf("result %v sniper %v", st["result"], st["sniper"])
	}
	store.Close()

	store = openTestStore(t, path)
	rec, _ := store.Game(m.id, 1)
	if rec.StartFEN == "" || !rec.sniper() || rec.RatingDiff[white] == nil {
		t.Fatalf("saved game: start %q sniper %v diff %v", rec.StartFEN, rec.sniper(), rec.RatingDiff)
	}
	// A Sniper Win moves ratings a quarter as far as a whole win.
	full := newPlayer.rate([]glickoResult{{newPlayer, 1}})
	want := int(math.Round(1500 + sniperWeight*(full.Rating-1500)))
	if a, _ := store.UserByID(ann.ID); int(math.Round(a.Rating)) != want {
		t.Errorf("ann's rating after a Sniper Win %.1f, want %d (a whole win: %.1f)", a.Rating, want, full.Rating)
	}
	// The replay starts from the setups.
	if r, err := buildReplay(rec); err != nil || r.Frames[0].Board[20] != 'R' {
		t.Errorf("replay: %v %v", err, r)
	}
}

func TestUnratedChoice(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	ann, _ := store.CreateUser("ann", []byte("x"))
	ben, _ := store.CreateUser("ben", []byte("x"))
	hub := NewHub(store)
	m := hub.Create(userPlayer(ann, "a"), Settings{Time: TimeControl{Base: 300}, Unrated: true})
	pb := userPlayer(ben, "b")
	m.join(newClient(pb.key), pb)
	if m.rated() {
		t.Error("an unrated game would be rated")
	}
}

func TestJumpKingMatch(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	hub := NewHub(store)
	m := hub.Create(guestPlayer("g1"), Settings{JumpKing: true})
	cw, cb := newClient("g1"), newClient("g2")
	m.join(cw, guestPlayer("g1"))
	m.join(cb, guestPlayer("g2"))
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	m.handle(cb, inMsg{Type: "move", UCI: "e7e5"})
	m.handle(cw, inMsg{Type: "move", UCI: "e1f3"}) // a knight's jump
	if n := len(m.game.History()); n != 3 {
		t.Fatalf("the king's jump wasn't played: %d plies", n)
	}
}

// setupOf is the setup part of a state message.
func setupOf(t *testing.T, s map[string]any) map[string]any {
	t.Helper()
	v, ok := s["setup"].(map[string]any)
	if !ok {
		t.Fatalf("no setup in %v", s)
	}
	return v
}
