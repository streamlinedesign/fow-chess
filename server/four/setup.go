package four

import (
	"errors"
	"math/rand/v2"
	"strings"
)

// Layout is how the players' pieces start: the usual rows, or one of the
// setup modes where each player arranges their own pieces first, without
// seeing anyone else's.
type Layout uint8

const (
	// Standard is the usual start.
	Standard Layout = iota
	// BackRow lets each player arrange their back row; pawns stay put.
	BackRow
	// Battleship lets each player place all sixteen pieces anywhere in their three rows.
	Battleship
)

// Rows is how many of a player's own rows a layout lets them use.
func (l Layout) Rows() int { return [...]int{0, 1, 3}[l] }

// army is each player's pieces.
const army = "KQRRBBNNPPPPPPPP"

// SetupLen is the length of a setup: a player's three own rows, back rank
// first, each from that player's left to right, as piece letters or '.'.
const SetupLen = 24

// StandardSetup is the usual arrangement as a setup string.
const StandardSetup = "RNBQKBNR" + "PPPPPPPP" + "........"

// CheckSetup reports whether s is a valid arrangement for layout l.
func CheckSetup(l Layout, s string) error {
	if l == Standard {
		return errors.New("this game has no setup")
	}
	if len(s) != SetupLen {
		return errors.New("a setup has 24 squares")
	}
	count := map[byte]int{}
	for i := range SetupLen {
		c := s[i]
		if c == '.' {
			continue
		}
		if !strings.ContainsRune("KQRBNP", rune(c)) {
			return errors.New("only your own pieces go in your setup")
		}
		row := i / 8
		switch {
		case l == BackRow && c == 'P' && row != 1:
			return errors.New("pawns stay on their usual squares")
		case l == BackRow && c != 'P' && row != 0:
			return errors.New("pieces go on your back row")
		case row >= l.Rows() && !(l == BackRow && row == 1):
			return errors.New("that square is outside your rows")
		}
		count[c]++
	}
	for _, k := range "KQRBNP" {
		if count[byte(k)] != strings.Count(army, string(k)) {
			return errors.New("place all of your pieces")
		}
	}
	return nil
}

// SetupPosition is the start with each player's own arrangement. Castling is
// allowed for a king and rook that start on their usual squares.
func SetupPosition(r Rules, setups [4]string) (Position, error) {
	p := Position{Turn: Red, EP: NoSquare, EPPawn: NoSquare, Rules: r}
	for _, c := range Colors {
		s := setups[c]
		if len(s) != SetupLen {
			return p, errors.New("a setup has 24 squares")
		}
		for i := range SetupLen {
			if k := kindFromLetter(s[i]); k != NoKind {
				p.Board[cell(c, i%8, i/8)] = NewPiece(c, k)
			}
		}
		king, kr, qr := castleSquares(c)
		if p.Board[king] == NewPiece(c, King) {
			if p.Board[kr] == NewPiece(c, Rook) {
				p.Castling |= 1 << (2 * c)
			}
			if p.Board[qr] == NewPiece(c, Rook) {
				p.Castling |= 2 << (2 * c)
			}
		}
	}
	return p, nil
}

// NewGameFromSetups starts a game from the players' arrangements.
func NewGameFromSetups(r Rules, setups [4]string) (*Game, error) {
	p, err := SetupPosition(r, setups)
	if err != nil {
		return nil, err
	}
	g := &Game{pos: p, setups: setups}
	return g, nil
}

// Setups is the arrangements the game started from, or all empty for the usual start.
func (g *Game) Setups() [4]string { return g.setups }

// BotSetup is an arrangement for a bot: the usual pieces on the back row in a
// random order, pawns in front.
func BotSetup(rng *rand.Rand) string {
	back := []byte("RNBQKBNR")
	rng.Shuffle(len(back), func(i, j int) { back[i], back[j] = back[j], back[i] })
	return string(back) + "PPPPPPPP" + "........"
}
