package main

import (
	"errors"
	"strconv"
	"strings"

	"chess/engine"
)

// setupMode is how players arrange their own pieces before the first move.
// Each player sets up without seeing the other's arrangement.
type setupMode struct {
	// rows is how many of a player's own ranks, counted from their back rank,
	// their pieces may go on. Zero means the usual starting position.
	rows int
	// pawnsFixed keeps every pawn on its usual square, so only the back row moves.
	pawnsFixed bool
}

// setupModes are the choices for Settings.Setup.
var setupModes = map[string]setupMode{
	"":           {},
	"backRow":    {rows: 1, pawnsFixed: true},
	"battleship": {rows: 4},
}

// army is the pieces each player has, as uppercase FEN letters.
const army = "KQRRBBNNPPPPPPPP"

// standardBoard is the usual starting position as a board string from a1 to h8.
const standardBoard = "RNBQKBNRPPPPPPPP" + "................................" + "pppppppprnbqkbnr"

// ownRow is how far square i is from color's back rank: 0 for the back rank itself.
func ownRow(color, i int) int {
	if color == white {
		return i / 8
	}
	return 7 - i/8
}

// ownSetup is color's pieces from the usual starting position, with every other square empty.
func ownSetup(color int) string {
	b := []byte(strings.Repeat(".", 64))
	for i := range b {
		if ownRow(color, i) < 2 {
			b[i] = standardBoard[i]
		}
	}
	return string(b)
}

// checkSetup reports whether board, 64 characters from a1 to h8, is a valid
// arrangement of color's own pieces under mode: all sixteen, each on a square
// the mode allows, and nothing else.
func checkSetup(mode setupMode, color int, board string) error {
	if mode.rows == 0 {
		return errors.New("this game has no setup")
	}
	if len(board) != 64 {
		return errors.New("a setup has 64 squares")
	}
	var count [128]int
	for i := range 64 {
		c := board[i]
		if c == '.' {
			continue
		}
		letter := c
		if color == black {
			letter = c - 'a' + 'A'
		}
		if !strings.ContainsRune("KQRBNP", rune(letter)) || (color == white) != (c >= 'A' && c <= 'Z') {
			return errors.New("only your own pieces go in your setup")
		}
		row := ownRow(color, i)
		switch {
		case letter == 'P' && mode.pawnsFixed:
			if row != 1 {
				return errors.New("pawns stay on their usual squares")
			}
		case mode.pawnsFixed && row != 0:
			return errors.New("pieces go on your back row")
		case row >= mode.rows:
			return errors.New("that square is outside your rows")
		}
		count[letter]++
	}
	for _, k := range "KQRBNP" {
		if count[k] != strings.Count(army, string(k)) {
			return errors.New("place all of your pieces")
		}
	}
	return nil
}

// setupFEN joins both players' setups into the starting position, White to
// move. Castling is allowed where a king and rook start on their usual squares.
func setupFEN(placed [2]string) string {
	board := []byte(placed[white])
	for i := range board {
		if placed[black][i] != '.' {
			board[i] = placed[black][i]
		}
	}
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

// sniperWin is whether a game was won on the winner's very first move, or
// before any move: a king that started under attack or already mated. It can
// only happen in setup modes, and counts for less rating than a whole game.
func sniperWin(result, reason string, plies int) bool {
	if reason != engine.Checkmate.String() && reason != engine.KingCaptured.String() {
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

// sniperWeight is how much of a whole game's rating change a Sniper Win moves.
const sniperWeight = 0.25
