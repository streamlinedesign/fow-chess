package bots

import "chess/engine"

// What the simulations taught the bots (about 650,000 bot games under fog;
// see the chess-bots repository): about four kings in five are taken by a
// piece their owner never saw, so a bot should keep the ways into its king
// covered, and a setup that guards the king almost never loses on the first
// move. Remembering where enemy pieces were last seen and closing in on the
// king were tried too, and didn't help a searching bot.

// exposureCost is what each way into the bot's king from the fog costs, in
// pawns. A variable so tests can tune it. At 0.6, Ramesses II with it scores
// about 0.59 against itself without it, over 2,000 fog games.
var exposureCost = 0.6

// exposure counts the ways into me's king from squares me can't see: each
// straight or diagonal line from the king that reaches the fog before it
// meets a piece, and each knight's jump away that's in the fog. Anything
// could be waiting there.
func exposure(p *engine.Position, me engine.Color, seen *[64]bool) int {
	king := p.KingSquare(me)
	if king == engine.NoSquare {
		return 0
	}
	n := 0
	for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		for s, ok := king.Offset(d[0], d[1]); ok; s, ok = s.Offset(d[0], d[1]) {
			if !seen[s] {
				n++
				break
			}
			if !p.Board[s].IsEmpty() {
				break
			}
		}
	}
	for _, d := range [8][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
		if s, ok := king.Offset(d[0], d[1]); ok && !seen[s] {
			n++
		}
	}
	return n
}

// GuardedKing reports whether a setup keeps its king safe from a first-move
// capture: the king on its owner's back two rows, and every forward line out
// of it blocked by its own pieces before it leaves their four rows. Knights
// can't reach it from outside them. own(row, file) maps the owner's rows (0 is
// their back row) to an index into board, 64 characters from a1 to h8 with
// '.' for an empty square, and row and file are where the king stands.
func GuardedKing(board string, row, file int, own func(row, file int) int) bool {
	if row > 1 {
		return false
	}
	for _, d := range [][2]int{{0, 1}, {1, 1}, {-1, 1}} { // forward lines; sideways and back stay inside
		for r, f := row+d[1], file+d[0]; f >= 0 && f <= 7; r, f = r+d[1], f+d[0] {
			if r > 3 {
				return false
			}
			if board[own(r, f)] != '.' {
				break
			}
		}
	}
	return true
}
