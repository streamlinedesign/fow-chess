package sim

import (
	"math/rand/v2"

	"chess/bots"
	"chess/engine"
)

// The site's own bots (server/bots) play here under their level names. They
// are handed the real position with the player's seen squares, and leave out
// what the fog hides themselves.
func init() {
	for _, level := range bots.Levels {
		Bots[string(level)] = func(rng *rand.Rand) Bot { return &siteBot{level, rng} }
	}
}

type siteBot struct {
	level bots.Level
	rng   *rand.Rand
}

func (b *siteBot) Move(v *View) engine.Move {
	var seen *[64]bool
	if v.Fog {
		seen = &v.Seen
	}
	m, _ := bots.Choose(b.level, v.pos, seen, b.rng)
	return m
}
