package sim

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"chess/engine"
)

// Config is how one game is played.
type Config struct {
	White, Black string
	// Mode is where the pieces start: "standard", "backRow" or "battleship".
	Mode string
	// Guarded setups, in Battleship, keep each king on its back two rows
	// with every line out of it blocked by its own pieces, as a careful
	// player would; otherwise pieces go anywhere at random.
	Guarded bool
	Fog     bool
	// Rules default to Eat the King (KingCapture). Turning KingCapture off
	// plays with check and checkmate, as the mods page can.
	Rules engine.Rules
	Seed  uint64
	// MaxPlies ends a game that runs this long as a draw. The fifty-move and
	// repetition rules nearly always end one first.
	MaxPlies int
	// Moves keeps every move in the record so the game can be replayed.
	Moves bool
}

// Record is one finished game, written as one line of JSON.
type Record struct {
	Seed  uint64 `json:"seed"`
	White string `json:"white"`
	Black string `json:"black"`
	Mode  string `json:"mode"`
	// Guarded is a Battleship game with guarded setups (see Config).
	Guarded bool   `json:"guarded,omitempty"`
	Fog     bool   `json:"fog"`
	Rules   string `json:"rules"`
	Result  string `json:"result"` // "1-0", "0-1" or "1/2-1/2"
	Method  string `json:"method"`
	Plies   int    `json:"plies"`
	// Sniper is a win on the winner's first move, as the site counts it.
	Sniper bool `json:"sniper,omitempty"`
	// Killer is the kind of piece that ate the king ("Q", "N", ...), and
	// KillerHidden whether the losing side could not see it just before.
	Killer       string `json:"killer,omitempty"`
	KillerHidden bool   `json:"killerHidden,omitempty"`
	// KingSteps is how many times the losing king moved.
	KingSteps int `json:"kingSteps,omitempty"`
	// Captures is how many pieces White and Black took.
	Captures [2]int `json:"captures"`
	// Material is White's and Black's material at the end, in pawns.
	Material [2]int `json:"material"`
	// FirstContact is the first ply on which either side saw an enemy piece
	// (with fog), and FirstCapture the first ply with a capture; -1 if never.
	FirstContact int `json:"firstContact"`
	FirstCapture int `json:"firstCapture"`
	// Vision is the share of the board each side saw, averaged over its turns.
	Vision   [2]float64 `json:"vision"`
	StartFEN string     `json:"start,omitempty"`
	Moves    string     `json:"moves,omitempty"`
}

// RulesName describes rules the way the mods page names them.
func RulesName(r engine.Rules) string {
	var parts []string
	if r.KingCapture {
		parts = append(parts, "eatKing")
	} else {
		parts = append(parts, "checkmate")
	}
	if r.JumpKing {
		parts = append(parts, "jumpKing")
	}
	if r.BackPawns {
		parts = append(parts, "backPawns")
	}
	if r.Hill {
		parts = append(parts, "hill")
	}
	return strings.Join(parts, "+")
}

// Play plays one game to the end.
func Play(c Config) (Record, error) {
	rng := rand.New(rand.NewPCG(c.Seed, 0x5eed))
	makers := [2]BotMaker{Bots[c.White], Bots[c.Black]}
	if makers[0] == nil || makers[1] == nil {
		return Record{}, fmt.Errorf("unknown bot %q or %q", c.White, c.Black)
	}
	bots := [2]Bot{makers[0](rand.New(rand.NewPCG(rng.Uint64(), 1))), makers[1](rand.New(rand.NewPCG(rng.Uint64(), 2)))}

	fen := engine.StartFEN
	if c.Mode != "" && c.Mode != "standard" {
		var err error
		if fen, err = RandomSetup(c.Mode, c.Rules.Hill, c.Guarded, rng); err != nil {
			return Record{}, err
		}
	}
	g, err := engine.NewGameFromFEN(fen, c.Rules)
	if err != nil {
		return Record{}, err
	}
	rec := Record{
		Seed: c.Seed, White: c.White, Black: c.Black, Mode: modeName(c.Mode), Guarded: c.Guarded && c.Mode == "battleship", Fog: c.Fog,
		Rules: RulesName(c.Rules), FirstContact: -1, FirstCapture: -1,
	}
	if fen != engine.StartFEN {
		rec.StartFEN = fen
	}
	maxPlies := c.MaxPlies
	if maxPlies <= 0 {
		maxPlies = 1000
	}
	var moves strings.Builder
	var visionSum [2]int
	var turns [2]int
	var kingSteps [2]int
	for !g.Over() && len(g.History()) < maxPlies {
		side := g.Turn()
		v := viewOf(g, side, c.Fog, c.Rules)
		n := 0
		for i, ok := range v.Seen {
			if ok {
				n++
				if rec.FirstContact < 0 && c.Fog && v.Board[i].Is(side.Other()) {
					rec.FirstContact = v.Ply
				}
			}
		}
		visionSum[side] += n
		turns[side]++

		m := bots[side].Move(v)
		pos := g.Position()
		if pos.Board[m.From].Kind() == engine.King {
			kingSteps[side]++
		}
		if pos.Board[m.To].Kind() == engine.King {
			rec.Killer = string(pos.Board[m.From].Kind().Letter())
			rec.KillerHidden = c.Fog && !pos.Vision(side.Other())[m.From]
		}
		pm, err := g.PlayUCI(m.UCI())
		if err != nil {
			return Record{}, fmt.Errorf("seed %d: %s played %s: %w", c.Seed, botName(c, side), m.UCI(), err)
		}
		if !pm.Captured.IsEmpty() {
			rec.Captures[side]++
			if rec.FirstCapture < 0 {
				rec.FirstCapture = len(g.History()) - 1
			}
		}
		if c.Moves {
			if moves.Len() > 0 {
				moves.WriteByte(' ')
			}
			moves.WriteString(m.UCI())
		}
	}

	rec.Plies = len(g.History())
	rec.Result, rec.Method = g.Outcome().String(), g.Method().String()
	if !g.Over() {
		rec.Result, rec.Method = engine.Drawn.String(), "move limit"
	}
	rec.Sniper = SniperWin(rec.Result, rec.Method, rec.Plies)
	switch rec.Result {
	case engine.WhiteWins.String():
		rec.KingSteps = kingSteps[engine.Black]
	case engine.BlackWins.String():
		rec.KingSteps = kingSteps[engine.White]
	}
	pos := g.Position()
	for _, pc := range pos.Board {
		if !pc.IsEmpty() && pc.Kind() != engine.King {
			rec.Material[pc.Color()] += value[pc.Kind()]
		}
	}
	for s := range 2 {
		if turns[s] > 0 {
			rec.Vision[s] = float64(visionSum[s]) / float64(turns[s]) / 64
		}
	}
	rec.Moves = moves.String()
	return rec, nil
}

func botName(c Config, side engine.Color) string {
	if side == engine.White {
		return c.White
	}
	return c.Black
}

func modeName(m string) string {
	if m == "" {
		return "standard"
	}
	return m
}

// SniperWin is a win on the winner's very first move, or before any move.
// It matches sniperWin in server/setup.go.
func SniperWin(result, method string, plies int) bool {
	if method != engine.Checkmate.String() && method != engine.KingCaptured.String() {
		return false
	}
	switch result {
	case engine.WhiteWins.String():
		return plies == 1
	case engine.BlackWins.String():
		return plies == 0 || plies == 2
	}
	return false
}

// army is each player's pieces as uppercase FEN letters.
const army = "KQRRBBNNPPPPPPPP"

// RandomSetup places both armies at random under a setup mode, the way the
// site's Back row and Battleship modes allow (server/setup.go), and returns
// the starting FEN with White to move. guarded applies to Battleship (see Config).
func RandomSetup(mode string, hill, guarded bool, rng *rand.Rand) (string, error) {
	var board [64]byte
	for i := range board {
		board[i] = '.'
	}
	for _, color := range []engine.Color{engine.White, engine.Black} {
		// ownSquare is the square on color's row r (0 = back rank), file f.
		ownSquare := func(r, f int) int {
			if color == engine.White {
				return r*8 + f
			}
			return (7-r)*8 + f
		}
		letter := func(k byte) byte {
			if color == engine.Black {
				return k | 0x20
			}
			return k
		}
		switch mode {
		case "backRow":
			back := []byte("KQRRBBNN")
			rng.Shuffle(len(back), func(i, j int) { back[i], back[j] = back[j], back[i] })
			for f := range 8 {
				board[ownSquare(0, f)] = letter(back[f])
				board[ownSquare(1, f)] = letter('P')
			}
		case "battleship":
			for {
				squares := rng.Perm(32)
				ok := true
				king := 0
				for i, k := range []byte(army) {
					sq := ownSquare(squares[i]/8, squares[i]%8)
					if k == 'K' && hill && engine.OnHill(engine.Square(sq)) {
						ok = false
					}
					if k == 'K' {
						king = squares[i]
					}
					board[sq] = letter(k)
				}
				if guarded && ok {
					ok = guardedKing(board, king, ownSquare)
				}
				if ok {
					break
				}
				for r := range 4 {
					for f := range 8 {
						board[ownSquare(r, f)] = '.'
					}
				}
			}
		default:
			return "", fmt.Errorf("unknown mode %q", mode)
		}
	}
	return setupFEN(board), nil
}

// guardedKing reports whether the king on own row king/8, file king%8, is on
// its back two rows and every line out of it meets one of its own pieces
// before it leaves its four rows. Knights can't reach it from outside them.
func guardedKing(board [64]byte, king int, ownSquare func(r, f int) int) bool {
	r, f := king/8, king%8
	if r > 1 {
		return false
	}
	for _, d := range [][2]int{{0, 1}, {1, 1}, {-1, 1}} { // forward lines; sideways and back stay inside
		for rr, ff := r+d[1], f+d[0]; ; rr, ff = rr+d[1], ff+d[0] {
			if ff < 0 || ff > 7 {
				break
			}
			if rr > 3 {
				return false
			}
			if board[ownSquare(rr, ff)] != '.' {
				break
			}
		}
	}
	return true
}

// setupFEN matches setupFEN in server/setup.go: castling is allowed where a
// king and rook start on their usual squares.
func setupFEN(board [64]byte) string {
	var fen strings.Builder
	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := range 8 {
			c := board[rank*8+file]
			if c == '.' {
				empty++
				continue
			}
			if empty > 0 {
				fen.WriteString(strconv.Itoa(empty))
				empty = 0
			}
			fen.WriteByte(c)
		}
		if empty > 0 {
			fen.WriteString(strconv.Itoa(empty))
		}
		if rank > 0 {
			fen.WriteByte('/')
		}
	}
	castling := ""
	for _, r := range []struct {
		flag         string
		king, rook   int
		kingC, rookC byte
	}{{"K", 4, 7, 'K', 'R'}, {"Q", 4, 0, 'K', 'R'}, {"k", 60, 63, 'k', 'r'}, {"q", 60, 56, 'k', 'r'}} {
		if board[r.king] == r.kingC && board[r.rook] == r.rookC {
			castling += r.flag
		}
	}
	if castling == "" {
		castling = "-"
	}
	return fen.String() + " w " + castling + " - 0 1"
}
