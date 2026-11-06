package bots

import "chess/engine"

// sight is engine.Position.Vision without the allocations, which matters
// because Solomon asks for it at every position it scores. Rules it doesn't
// handle itself (jester pawns, jumping kings, castling) it leaves to the
// engine, piece by piece. TestSightMatchesVision keeps the two in step.
func sight(p *engine.Position, c engine.Color) (seen [64]bool) {
	if p.Outpost(c) != engine.NoSquare {
		for i := range seen {
			seen[i] = true
		}
		return seen
	}
	plain := plainPawns(p)
	for i, pc := range p.Board {
		if !pc.Is(c) {
			continue
		}
		sq := engine.Square(i)
		seen[i] = true
		f, r := i&7, i>>3
		switch pc.Kind() {
		case engine.Pawn:
			if !plain {
				return p.Vision(c) // rare: let the engine do it all
			}
			dir, start := 1, 1
			if c == engine.Black {
				dir, start = -1, 6
			}
			if r+dir >= 0 && r+dir < 8 {
				seen[i+8*dir] = true
				if r == start {
					seen[i+16*dir] = true
				}
				for _, df := range [2]int{-1, 1} {
					if f+df < 0 || f+df > 7 {
						continue
					}
					to := i + 8*dir + df
					if t := p.Board[to]; t.Is(c.Other()) || t.IsEmpty() && engine.Square(to) == p.EP && c == p.Turn {
						seen[to] = true
					}
				}
			}
		case engine.Knight:
			for _, to := range knightTargets[i] {
				if !p.Board[to].Is(c) {
					seen[to] = true
				}
			}
		case engine.King:
			if p.JumpKing || p.Castling != 0 {
				for _, m := range p.PieceMoves(sq) {
					seen[m.To] = true
				}
				continue
			}
			for _, to := range kingTargets[i] {
				if !p.Board[to].Is(c) {
					seen[to] = true
				}
			}
		default:
			k := pc.Kind()
			for d, dir := range dirs {
				if d < 4 && k == engine.Bishop || d >= 4 && k == engine.Rook {
					continue
				}
				for ff, rr := f+dir[0], r+dir[1]; ff >= 0 && ff < 8 && rr >= 0 && rr < 8; ff, rr = ff+dir[0], rr+dir[1] {
					to := rr*8 + ff
					t := p.Board[to]
					if t.Is(c) {
						break
					}
					seen[to] = true
					if !t.IsEmpty() {
						break
					}
				}
			}
		}
	}
	return seen
}

// dirs are the rook's four directions, then the bishop's.
var dirs = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

var knightTargets, kingTargets = func() (n, k [64][]int) {
	for i := range 64 {
		f, r := i&7, i>>3
		for _, d := range [8][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
			if ff, rr := f+d[0], r+d[1]; ff >= 0 && ff < 8 && rr >= 0 && rr < 8 {
				n[i] = append(n[i], rr*8+ff)
			}
		}
		for _, d := range dirs {
			if ff, rr := f+d[0], r+d[1]; ff >= 0 && ff < 8 && rr >= 0 && rr < 8 {
				k[i] = append(k[i], rr*8+ff)
			}
		}
	}
	return
}()
