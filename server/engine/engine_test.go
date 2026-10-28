package engine

import (
	"strings"
	"testing"
)

func perft(p *Position, depth int) int {
	moves := p.LegalMoves()
	if depth == 1 {
		return len(moves)
	}
	n := 0
	for _, m := range moves {
		next := p.Apply(m)
		n += perft(&next, depth-1)
	}
	return n
}

// Reference counts from https://www.chessprogramming.org/Perft_Results. These
// positions are built to hit castling, en passant, promotion and pin edge cases.
var perftCases = []struct {
	name   string
	fen    string
	counts []int // by depth, starting at 1
}{
	{"start", StartFEN, []int{20, 400, 8902, 197281, 4865609}},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", []int{48, 2039, 97862, 4085603}},
	{"position 3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", []int{14, 191, 2812, 43238, 674624}},
	{"position 4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", []int{6, 264, 9467, 422333}},
	{"position 4 mirrored", "r2q1rk1/pP1p2pp/Q4n2/bbp1p3/Np6/1B3NBn/pPPP1PPP/R3K2R b KQ - 0 1", []int{6, 264, 9467, 422333}},
	{"position 5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", []int{44, 1486, 62379, 2103487}},
	{"position 6", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", []int{46, 2079, 89890, 3894594}},
}

func TestPerft(t *testing.T) {
	for _, c := range perftCases {
		t.Run(c.name, func(t *testing.T) {
			p, err := ParseFEN(c.fen)
			if err != nil {
				t.Fatal(err)
			}
			for i, want := range c.counts {
				depth := i + 1
				if testing.Short() && want > 100000 {
					break
				}
				if got := perft(&p, depth); got != want {
					t.Fatalf("depth %d: got %d, want %d", depth, got, want)
				}
			}
		})
	}
}

func TestFENRoundTrip(t *testing.T) {
	for _, c := range perftCases {
		p, err := ParseFEN(c.fen)
		if err != nil {
			t.Fatal(err)
		}
		if got := p.FEN(); got != c.fen {
			t.Errorf("got %q, want %q", got, c.fen)
		}
	}
	for _, bad := range []string{"", "8/8/8 w - -", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNX w KQkq - 0 1", StartFEN[:len(StartFEN)-12] + " x KQkq - 0 1"} {
		if _, err := ParseFEN(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func play(t *testing.T, g *Game, moves string) []string {
	t.Helper()
	var sans []string
	for _, uci := range strings.Fields(moves) {
		pm, err := g.PlayUCI(uci)
		if err != nil {
			t.Fatalf("%s: %v (fen %s)", uci, err, g.pos.FEN())
		}
		sans = append(sans, pm.SAN)
	}
	return sans
}

func TestSAN(t *testing.T) {
	g := NewGame()
	got := play(t, g, "e2e4 a7a6 e4e5 d7d5 e5d6 b7b6 g1f3 a6a5 f1e2 a5a4 e1g1 a4a3 d6c7 b6b5 c7b8q")
	want := "e4 a6 e5 d5 exd6 b6 Nf3 a5 Be2 a4 O-O a3 dxc7 b5 cxb8=Q"
	if strings.Join(got, " ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " "), want)
	}

	// Disambiguation by file, by rank, and by both.
	for _, c := range []struct{ fen, uci, san string }{
		{"4k3/8/8/8/8/8/8/1N2KN2 w - - 0 1", "b1d2", "Nbd2"},
		{"4k3/8/8/8/R7/8/8/R3K3 w - - 0 1", "a1a2", "R1a2"},
		{"4k3/8/8/8/8/2Q1Q3/8/2Q1K3 w - - 0 1", "e3d2", "Qed2"},
		{"1k6/8/8/8/4Q2Q/8/8/K6Q w - - 0 1", "h4e1", "Qh4e1"},
		{"4k3/8/8/8/8/8/8/R3K2R w KQ - 0 1", "e1c1", "O-O-O"},
		{"7k/8/8/8/8/8/8/K5R1 w - - 0 1", "g1h1", "Rh1+"},
		{"6k1/5ppp/8/8/8/8/8/K3R3 w - - 0 1", "e1e8", "Re8#"},
	} {
		p, err := ParseFEN(c.fen)
		if err != nil {
			t.Fatal(err)
		}
		m, ok := p.FindMove(c.uci)
		if !ok {
			t.Fatalf("%s not legal in %s", c.uci, c.fen)
		}
		if got := p.SAN(m); got != c.san {
			t.Errorf("%s: got %s, want %s", c.fen, got, c.san)
		}
	}
}

func TestOutcomes(t *testing.T) {
	cases := []struct {
		name, fen, moves string
		outcome          Outcome
		method           Method
	}{
		{"fool's mate", StartFEN, "f2f3 e7e5 g2g4 d8h4", BlackWins, Checkmate},
		{"stalemate", "7k/5Q2/8/8/8/8/8/K7 w - - 0 1", "f7g6", Drawn, Stalemate},
		{"insufficient after capture", "8/8/2n5/4k3/1Q6/8/8/4K3 b - - 0 1", "c6b4", Drawn, InsufficientMaterial},
		{"king vs king", "8/8/8/4k3/8/8/3q4/4K3 w - - 0 1", "e1d2", Drawn, InsufficientMaterial},
		{"same-color bishops", "1b6/8/8/4k3/8/8/3r4/2B1K3 w - - 0 1", "e1d2", Drawn, InsufficientMaterial},
		{"threefold repetition", StartFEN, "g1f3 g8f6 f3g1 f6g8 g1f3 g8f6 f3g1 f6g8", Drawn, Repetition},
		{"fifty-move rule", "4k3/8/8/8/8/8/8/R3K3 w - - 99 80", "a1a2", Drawn, FiftyMoves},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g, err := NewGameFromFEN(c.fen)
			if err != nil {
				t.Fatal(err)
			}
			play(t, g, c.moves)
			if g.Outcome() != c.outcome || g.Method() != c.method {
				t.Errorf("got %v by %v, want %v by %v", g.Outcome(), g.Method(), c.outcome, c.method)
			}
		})
	}
}

func TestRejectsBadMoves(t *testing.T) {
	g := NewGame()
	for _, uci := range []string{"e2e5", "e7e5", "e1g1", "a1a3", "zz", "e2e4x"} {
		if _, err := g.PlayUCI(uci); err == nil {
			t.Errorf("%s should be rejected", uci)
		}
	}
	// A pinned piece can't move, and promotion needs a piece letter.
	g, _ = NewGameFromFEN("4k3/4r3/8/8/8/8/4N3/4K3 w - - 0 1")
	if _, err := g.PlayUCI("e2c3"); err == nil {
		t.Error("pinned knight moved")
	}
	g, _ = NewGameFromFEN("4k3/P7/8/8/8/8/8/4K3 w - - 0 1")
	if _, err := g.PlayUCI("a7a8"); err == nil {
		t.Error("promotion without a piece was accepted")
	}
	g.Resign(White)
	if _, err := g.PlayUCI("a7a8q"); err != ErrGameOver {
		t.Errorf("move after resignation: %v", err)
	}
}

func BenchmarkPerftStart4(b *testing.B) {
	p := StartPosition()
	for b.Loop() {
		perft(&p, 4)
	}
}

func visionString(v [64]bool) string {
	b := make([]byte, 0, 72)
	for rank := 7; rank >= 0; rank-- {
		for file := 0; file < 8; file++ {
			if v[SquareAt(file, rank)] {
				b = append(b, 'x')
			} else {
				b = append(b, '.')
			}
		}
		b = append(b, '/')
	}
	return string(b[:len(b)-1])
}

func TestVision(t *testing.T) {
	cases := []struct {
		name, fen string
		viewer    Color
		want      string // ranks 8 to 1
	}{
		{"start", StartFEN, White, "......../......../......../......../xxxxxxxx/xxxxxxxx/xxxxxxxx/xxxxxxxx"},
		{"start for black", StartFEN, Black, "xxxxxxxx/xxxxxxxx/xxxxxxxx/xxxxxxxx/......../......../......../........"},
		// Blocked pawn still sees ahead; diagonals only with a capture; rook sees up to and including blockers.
		{"pawn and rook", "4k3/8/8/3p4/3P4/8/8/R3K3 w - - 0 1", White,
			"x......./x......./x......./x..x..../x..x..../x......./x..xxx../xxxxxx.."},
		{"pawn capture", "4k3/8/8/2p5/3P4/8/8/4K3 w - - 0 1", White,
			"......../......../......../..xx..../...x..../......../...xxx../...xxx.."},
		// The en passant square only counts for the side that may capture onto it.
		{"en passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", White,
			"......../......../...xx.../....x.../......../......../...xxx../...xxx.."},
		{"no en passant for the other side", "4k3/8/8/8/3pP3/8/8/4K3 w - e3 0 1", Black,
			"...xxx../...xxx../......../......../...x..../...x..../......../........"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := ParseFEN(c.fen)
			if err != nil {
				t.Fatal(err)
			}
			if got := visionString(p.Vision(c.viewer)); got != c.want {
				t.Errorf("\n got %s\nwant %s", got, c.want)
			}
		})
	}
}

func TestKingCapture(t *testing.T) {
	r := Rules{KingCapture: true}
	// Moving into attack is allowed, and capturing the king wins.
	g, _ := NewGameFromFEN("4k3/8/8/8/8/8/3r4/4K3 w - - 0 1", r)
	if _, err := g.PlayUCI("e1d1"); err != nil {
		t.Fatalf("king should be allowed into attack: %v", err)
	}
	if _, err := g.PlayUCI("d2d1"); err != nil {
		t.Fatal(err)
	}
	if g.Outcome() != BlackWins || g.Method() != KingCaptured {
		t.Errorf("got %v by %v", g.Outcome(), g.Method())
	}

	// A position that is mate with check rules is just a position without them.
	g, _ = NewGameFromFEN(StartFEN, r)
	sans := play(t, g, "f2f3 e7e5 g2g4 d8h4")
	if g.Over() || sans[3] != "Qh4" {
		t.Errorf("fool's mate without check: over=%v san=%s", g.Over(), sans[3])
	}
	play(t, g, "a2a3 h4e1")
	if g.Outcome() != BlackWins || g.Method() != KingCaptured {
		t.Errorf("got %v by %v", g.Outcome(), g.Method())
	}

	// Castling through an attacked square is allowed without check.
	g, _ = NewGameFromFEN("4k3/8/8/8/8/8/5r2/4K2R w K - 0 1", r)
	if _, err := g.PlayUCI("e1g1"); err != nil {
		t.Errorf("castle through attack: %v", err)
	}
}

func TestTimeOut(t *testing.T) {
	cases := []struct {
		name   string
		fen    string
		want   Outcome
		method Method
	}{
		{"white flags with black able to win", "4k3/8/8/8/8/8/4P3/4K2q w - - 0 1", BlackWins, Timeout},
		{"black flags against a lone king", "4k3/8/8/8/8/8/8/4K3 b - - 0 1", Drawn, InsufficientMaterial},
		{"black flags against king and knight", "4k2r/8/8/8/8/8/8/4K1N1 b - - 0 1", Drawn, TimeoutVsInsufficientMaterial},
		{"black flags against king and pawn", "4k2r/8/8/8/8/8/4P3/4K3 b - - 0 1", WhiteWins, Timeout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g, err := NewGameFromFEN(c.fen)
			if err != nil {
				t.Fatal(err)
			}
			g.TimeOut()
			if g.Outcome() != c.want || g.Method() != c.method {
				t.Errorf("got %v by %v, want %v by %v", g.Outcome(), g.Method(), c.want, c.method)
			}
		})
	}
}
