package bots

import "chess/engine"

// plainPawns reports whether pawns move the usual way in p. It's the one
// place the bots name a rule that differs between fow-chess's engine and the
// copy in chess-bots, so this file is the one to edit when copying bots/.
func plainPawns(p *engine.Position) bool { return !p.JesterPawns }
