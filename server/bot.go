package main

import (
	"math/rand/v2"
	"strings"
	"time"

	"chess/bots"
	"chess/engine"
)

// A bot holds a seat like any player, with the key "bot:<level>". It has no
// connection: the match pokes it whenever something changes, and it answers
// from its own view of the board.

// botColor is the seat the bot holds, or -1.
func (m *Match) botColor() int {
	if m.settings.Bot == "" {
		return -1
	}
	return m.colorOf("bot:" + m.settings.Bot)
}

// pokeBot has the bot do whatever the match needs from it: set up its
// pieces, accept a rematch, or think about a move. Callers hold m.mu.
func (m *Match) pokeBot() {
	c := m.botColor()
	if c < 0 || m.seats[black] == "" {
		return
	}
	now := time.Now()
	switch {
	case m.game.Over():
		if m.rematch[1-c] && !m.rematch[c] {
			m.rematch[c] = true
			m.startRematch(now)
			m.save()
			m.pokeBot()
		}
	case !m.setUp:
		if m.placed[c] == "" {
			m.placed[c] = randomSetup(setupModes[m.settings.Setup], c, m.settings.Hill)
			if m.placed[white] != "" && m.placed[black] != "" {
				m.startSetUp(now)
				m.pokeBot()
			}
		}
	case int(m.game.Turn()) == c && !m.botThinking:
		m.botThinking = true
		pos := m.game.Position()
		var seen *[64]bool
		if m.settings.Fog {
			v := pos.Vision(engine.Color(c))
			seen = &v
		}
		level := bots.Level(m.settings.Bot)
		round, plies := m.round, len(m.game.History())
		time.AfterFunc(m.botDelay(c, now), func() {
			mv, ok := bots.Choose(level, pos, seen, nil)
			m.mu.Lock()
			defer m.mu.Unlock()
			m.botThinking = false
			if !ok || m.round != round || len(m.game.History()) != plies || !m.playMove(c, mv.UCI(), time.Now()) {
				return
			}
			m.save()
			m.broadcast()
		})
	}
}

// botDelay is how long the bot waits before moving: long enough to feel like
// a player, never a real share of its clock. Callers hold m.mu.
func (m *Match) botDelay(c int, now time.Time) time.Duration {
	d := 400*time.Millisecond + time.Duration(rand.IntN(900))*time.Millisecond
	if m.running() {
		d = min(d, m.left(c, now)/30)
	}
	return max(d, 50*time.Millisecond)
}

// randomSetup arranges color's pieces at random within what mode allows.
func randomSetup(mode setupMode, color int, hill bool) string {
	var squares []int
	for i := range 64 {
		if ownRow(color, i) < mode.rows && !(mode.pawnsFixed && ownRow(color, i) == 1) {
			squares = append(squares, i)
		}
	}
	for {
		b := []byte(strings.Repeat(".", 64))
		pieces := []byte(army)
		if mode.pawnsFixed {
			pieces = pieces[:8] // the pawns stay where they are
			for i := range 64 {
				if ownRow(color, i) == 1 {
					b[i] = standardBoard[i]
				}
			}
		}
		rand.Shuffle(len(squares), func(i, j int) { squares[i], squares[j] = squares[j], squares[i] })
		for i, pc := range pieces {
			if color == black {
				pc = pc - 'A' + 'a'
			}
			b[squares[i]] = pc
		}
		if checkSetup(mode, color, string(b), hill) == nil {
			return string(b)
		}
	}
}
