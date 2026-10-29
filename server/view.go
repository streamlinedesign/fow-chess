package main

import (
	"slices"
	"strings"

	"chess/engine"
)

// Settings are chosen when a match is created, for play testing different rules.
type Settings struct {
	// Fog hides every square the viewer's pieces can't see.
	Fog bool `json:"fog"`
	// KingCapture plays without check: win by capturing the king.
	KingCapture bool `json:"kingCapture"`
	// Time is the clock. The zero value is no clock, which is also what games
	// saved before clocks existed load as.
	Time TimeControl `json:"time"`
	// Setup is how the pieces start: "" for the usual position, or one of the
	// setup modes, where each player arranges their own pieces first.
	Setup string `json:"setup,omitempty"`
	// JumpKing lets kings also jump like a knight.
	JumpKing bool `json:"jumpKing,omitempty"`
	// BackPawns lets pawns also step one square straight back, and see behind them.
	BackPawns bool `json:"backPawns,omitempty"`
	// Hill is King of the Hill: a king on one of the four centre squares wins.
	Hill bool `json:"hill,omitempty"`
	// Unrated keeps a game that could be rated from counting.
	Unrated bool `json:"unrated,omitempty"`
	// Mods is a game from the mods page: any of the options above, any clock,
	// never ranked.
	Mods bool `json:"mods,omitempty"`
}

func (s Settings) rules() engine.Rules {
	return engine.Rules{KingCapture: s.KingCapture, JumpKing: s.JumpKing, BackPawns: s.BackPawns, Hill: s.Hill}
}

func (s Settings) valid() bool {
	_, ok := setupModes[s.Setup]
	return (s.Time.valid() || (s.Mods && s.Time.modsValid())) && ok
}

// normalize makes new settings the one game the site plays: fog of war, win
// by eating the king. A mods game keeps its options but is never ranked.
// Saved games keep whatever they were played with.
func (s *Settings) normalize() {
	if s.Mods {
		s.Unrated = true
		return
	}
	s.Fog, s.KingCapture = true, true
	s.JumpKing, s.BackPawns, s.Hill = false, false, false
}

// startFrom is the FEN a game starts from: the usual position, or the players' setups.
func startFrom(fen string) string {
	if fen == "" {
		return engine.StartFEN
	}
	return fen
}

// TimeControl is a clock setting: Base minutes each plus Inc seconds per move,
// or PerMove seconds for every move (Daily). All zero means no clock.
type TimeControl struct {
	Base    int `json:"base"`    // seconds per player
	Inc     int `json:"inc"`     // seconds added after each move
	PerMove int `json:"perMove"` // seconds for each move; the clock resets every move
}

// timeControls are the presets players can pick from. Must match TIME_GROUPS in web/src/settings.ts.
var timeControls = []TimeControl{
	// Blitz
	{Base: 300}, {Base: 180, Inc: 2},
	// Rapid
	{Base: 600},
	// Tournament
	{Base: 5400, Inc: 30},
	// Daily and weekly: the clock resets after every move
	{PerMove: 24 * 60 * 60}, {PerMove: 7 * 24 * 60 * 60},
}

// modTimeControls are the extra clocks a mods game can have, no clock among
// them. Must match MOD_TIMES in web/src/settings.ts.
var modTimeControls = []TimeControl{
	{},
	{Base: 60}, {Base: 120, Inc: 1},
	{Base: 180}, {Base: 300, Inc: 3},
	{Base: 600, Inc: 5}, {Base: 900, Inc: 10},
	{Base: 1800}, {Base: 1800, Inc: 20},
}

// valid is checked only when a game or an offer is created. Saved games keep
// whatever clock they were made with, including the presets no longer offered.
func (t TimeControl) valid() bool {
	return slices.Contains(timeControls, t)
}

func (t TimeControl) modsValid() bool {
	return slices.Contains(modTimeControls, t)
}

// clocked is false for No clock games, which wait forever and are never rated.
func (t TimeControl) clocked() bool { return t != TimeControl{} }

// boardView encodes what a viewer can see as 64 characters from a1 to h8:
// FEN letters for pieces, '.' for an empty square, and '?' for a square the
// viewer can't see. A nil visible mask shows everything.
func boardView(pos *engine.Position, visible *[64]bool) string {
	b := make([]byte, 64)
	for i, pc := range pos.Board {
		switch {
		case visible != nil && !visible[i]:
			b[i] = '?'
		case pc.IsEmpty():
			b[i] = '.'
		default:
			b[i] = pc.Letter()
		}
	}
	return string(b)
}

type lastMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// gameView is the part of the state message that depends on who is looking.
type gameView struct {
	Board string    `json:"board"`
	Turn  string    `json:"turn"`
	Ply   int       `json:"ply"`
	Legal []string  `json:"legal"`
	Last  *lastMove `json:"last"`
	Check string    `json:"check,omitempty"`
	// Danger is the player's king square when a piece they can see attacks it,
	// and Unsafe the squares their king could step to that a seen piece attacks.
	// Both come from the player's own view, so they give nothing away.
	Danger  string   `json:"danger,omitempty"`
	Unsafe  []string `json:"unsafe,omitempty"`
	History []string `json:"history"` // SAN, or "?" for moves the viewer may not know
	// Captured lists the pieces each side has taken, as FEN letters, most valuable first.
	// Players know all of it even under fog: they saw what they took and see what they lost.
	Captured map[string]string `json:"captured"`
	// Behind is how many of the moves played a spectator isn't shown yet.
	Behind int `json:"behind,omitempty"`
}

// captured lists the pieces each side has taken, most valuable first.
func captured(hist []engine.PlayedMove) map[string]string {
	var by [2][]engine.Piece
	for i, pm := range hist {
		if !pm.Captured.IsEmpty() {
			by[i%2] = append(by[i%2], pm.Captured)
		}
	}
	out := map[string]string{}
	for c, pcs := range by {
		slices.SortStableFunc(pcs, func(a, b engine.Piece) int { return int(b.Kind()) - int(a.Kind()) })
		letters := make([]byte, len(pcs))
		for i, pc := range pcs {
			letters[i] = pc.Letter()
		}
		out[engine.Color(c).String()] = string(letters)
	}
	return out
}

// viewFor builds the game as seen by viewer (-1 for everything, which is how
// spectators see a game without fog). Legal moves are only sent to the player
// whose turn it is, so the browser never has to know the rules.
//
// With fog, a player sees only what their pieces see and gets no hint of the
// opponent's moves: no last-move highlight, no notation, and no check marks on
// their own moves. Once the game is over everyone sees everything.
func viewFor(g *engine.Game, viewer int, playable, fog bool) gameView {
	pos := g.Position()
	hist := g.History()
	n := len(hist)
	v := gameView{
		Turn:     pos.Turn.String(),
		Ply:      n,
		Legal:    []string{},
		History:  make([]string, n),
		Captured: captured(hist),
	}
	if playable && !g.Over() && viewer == int(pos.Turn) {
		for _, m := range g.LegalMoves() {
			v.Legal = append(v.Legal, m.UCI())
		}
	}

	if viewer >= 0 && !g.Over() {
		var seen *[64]bool
		if fog {
			vision := pos.Vision(engine.Color(viewer))
			seen = &vision
		}
		v.Danger, v.Unsafe = kingDanger(&pos, engine.Color(viewer), seen)
	}

	if !fog || g.Over() || viewer < 0 {
		v.Board = boardView(&pos, nil)
		for i, pm := range hist {
			v.History[i] = pm.SAN
		}
		if n > 0 {
			v.Last = &lastMove{From: hist[n-1].Move.From.String(), To: hist[n-1].Move.To.String()}
		}
		if pos.InCheck() {
			v.Check = pos.KingSquare(pos.Turn).String()
		}
		return v
	}

	// Ply i was played by white when i is even (games start from the initial position).
	mover := func(i int) int { return i % 2 }

	vision := pos.Vision(engine.Color(viewer))
	v.Board = boardView(&pos, &vision)
	for i, pm := range hist {
		if mover(i) == viewer {
			v.History[i] = strings.TrimRight(pm.SAN, "+#")
		} else {
			v.History[i] = "?"
		}
	}
	if n > 0 && mover(n-1) == viewer {
		v.Last = &lastMove{From: hist[n-1].Move.From.String(), To: hist[n-1].Move.To.String()}
	}
	// You know when you're in check, since it limits your moves; never the reverse.
	if viewer == int(pos.Turn) && pos.InCheck() {
		v.Check = pos.KingSquare(pos.Turn).String()
	}
	return v
}

// kingDanger says whether a piece the player can see attacks their king, and
// which of the king's moves land on a square a seen piece attacks. Pieces the
// fog hides count as blockers but never as attackers, so the answer only uses
// what is on the player's own board. A nil seen means everything is visible.
func kingDanger(pos *engine.Position, us engine.Color, seen *[64]bool) (danger string, unsafe []string) {
	king := pos.KingSquare(us)
	if king == engine.NoSquare {
		return "", nil
	}
	attacks := func(p *engine.Position, target engine.Square) bool {
		for sq := range p.Board {
			pc := p.Board[sq]
			if pc.IsEmpty() || pc.Color() == us || (seen != nil && !seen[sq]) {
				continue
			}
			for _, m := range p.PieceMoves(engine.Square(sq)) {
				if m.To == target {
					return true
				}
			}
		}
		return false
	}
	if attacks(pos, king) {
		danger = king.String()
	}
	if pos.Turn == us {
		for _, m := range pos.LegalMoves() {
			if m.From != king {
				continue
			}
			after := pos.Apply(m)
			if attacks(&after, m.To) {
				unsafe = append(unsafe, m.To.String())
			}
		}
	}
	return danger, unsafe
}

// spectatorDelay is how many moves (by either side) spectators of a fog game
// are kept behind, so nobody watching can tell a player what the fog hides
// from them right now.
const spectatorDelay = 6

// spectatorView is the game as a spectator sees it. Without fog, or once the
// game is over, that's everything. In a fog game still being played, it's what
// side saw spectatorDelay moves ago; the turn and move count stay current, so
// the page can say whose move it is.
func spectatorView(g *engine.Game, rules engine.Rules, side int, fog bool) gameView {
	if !fog || g.Over() {
		return viewFor(g, -1, false, fog)
	}
	hist := g.History()
	shown := max(0, len(hist)-spectatorDelay)
	past, _ := engine.NewGameFromFEN(g.StartFEN(), rules)
	for _, pm := range hist[:shown] {
		past.PlayUCI(pm.Move.UCI())
	}
	v := viewFor(past, side, false, true)
	v.Turn = g.Turn().String()
	v.Ply = len(hist)
	v.Behind = len(hist) - shown
	return v
}
