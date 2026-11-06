package main

import (
	"chess/bots"
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
	// JesterPawns lets pawns also step and take one square in any direction, like a king.
	JesterPawns bool `json:"jesterPawns,omitempty"`
	// Hill is King of the Hill: a king on one of the four centre squares wins.
	Hill bool `json:"hill,omitempty"`
	// CenterVision gives the whole board to a player with a piece on the
	// centre of the opponent's half: "pawn" counts only pawns, "any" any piece.
	CenterVision string `json:"centerVision,omitempty"`
	// Sync is Sync Move: both players send a move each round and both happen
	// at once (see engine.Game.PlayRound). Mods only, and always Eat the King.
	Sync bool `json:"sync,omitempty"`
	// Unrated keeps a game from counting. Players can't ask for it: every
	// game of the one game is ranked when it can be, and normalize sets it
	// for bot and mods games, which never are.
	Unrated bool `json:"unrated,omitempty"`
	// Mods is a game from the mods page: any of the options above, any clock,
	// never ranked.
	Mods bool `json:"mods,omitempty"`
	// Bot is the computer opponent's level when one seat is a bot. Never ranked.
	Bot string `json:"bot,omitempty"`
	// Tournament is the arena this game is part of, if any.
	Tournament int64 `json:"tournament,omitempty"`
	// Demo is the home page's warm-up game against a bot (Solomon from the home
	// page): no clock, kept only in memory, never listed or ranked.
	Demo bool `json:"demo,omitempty"`
}

// demoBot is who the home page's demo board plays unless the player picks another bot.
const demoBot = bots.Solomon

func (s Settings) rules() engine.Rules {
	return engine.Rules{KingCapture: s.KingCapture, JumpKing: s.JumpKing, Hill: s.Hill,
		CenterVision: centerVisions[s.CenterVision], JesterPawns: s.JesterPawns, Sync: s.Sync}
}

// centerVisions are the CenterVision settings.
var centerVisions = map[string]engine.CenterVision{"": engine.CenterOff, "pawn": engine.CenterPawn, "any": engine.CenterAny}

func (s Settings) valid() bool {
	_, ok := setupModes[s.Setup]
	_, center := centerVisions[s.CenterVision]
	return center && (s.Time.valid() || (s.Mods && s.Time.modsValid()) || (s.Demo && !s.Time.clocked())) && ok &&
		(s.Bot == "" || bots.Level(s.Bot).Valid())
}

// normalize makes new settings the one game the site plays: fog of war with
// centre vision (any piece), win by eating the king, ranked whenever both players are signed in. A mods game
// keeps its options but is never ranked, and neither is a bot game. Saved
// games keep whatever they were played with.
func (s *Settings) normalize() {
	s.Unrated = false
	if s.Demo {
		bot := bots.Level(s.Bot)
		if !bot.Valid() {
			bot = demoBot
		}
		*s = Settings{Demo: true, Bot: string(bot)}
	}
	if s.Bot != "" {
		s.Unrated = true
	}
	if s.Mods {
		s.Unrated = true
		if s.Sync {
			s.KingCapture = true // there's no check when both sides move at once
		}
		return
	}
	s.Sync = false
	s.Fog, s.KingCapture, s.CenterVision = true, true, "any"
	s.JumpKing, s.JesterPawns, s.Hill = false, false, false
	// Eat the King is played from the usual position; Back row and
	// Battleship are mods.
	s.Setup = ""
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
	// Also is the other move of the last Sync Move round, when the viewer may see it.
	Also  *lastMove `json:"also,omitempty"`
	Check string    `json:"check,omitempty"`
	// Danger is the player's king square when any enemy piece attacks it, even
	// one the fog hides (the player learns the king is attacked, not from
	// where), and Unsafe the squares their king could step to that a seen
	// piece attacks, which come from the player's own view.
	Danger string `json:"danger,omitempty"`
	// DangerHidden is set when the king is attacked only from the fog.
	DangerHidden bool     `json:"dangerHidden,omitempty"`
	Unsafe       []string `json:"unsafe,omitempty"`
	// Reach is where each enemy piece the viewer can see could move, worked out
	// on the board as the viewer sees it, so it tells them nothing new.
	Reach   map[string][]string `json:"reach,omitempty"`
	History []string            `json:"history"` // SAN, or "?" for moves the viewer may not know
	// Captured lists the pieces each side has taken, as FEN letters, most valuable first.
	// Players know all of it even under fog: they saw what they took and see what they lost.
	Captured map[string]string `json:"captured"`
	// FullVision is set while the viewer sees the whole board from an outpost.
	FullVision bool `json:"fullVision,omitempty"`
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
	if playable && !g.Over() && viewer >= 0 && (pos.Sync || viewer == int(pos.Turn)) {
		for _, m := range pos.MovesFor(engine.Color(viewer)) {
			v.Legal = append(v.Legal, m.UCI())
		}
	}

	if viewer >= 0 && !g.Over() {
		var seen *[64]bool
		if fog {
			vision := pos.Vision(engine.Color(viewer))
			seen = &vision
		}
		v.Danger, v.DangerHidden, v.Unsafe = kingDanger(&pos, engine.Color(viewer), seen)
		if playable {
			v.Reach = enemyReach(&pos, engine.Color(viewer), seen)
		}
	}

	if !fog || g.Over() || viewer < 0 {
		v.Board = boardView(&pos, nil)
		for i, pm := range hist {
			v.History[i] = pm.SAN
		}
		if n > 0 && !hist[n-1].IsPass() {
			v.Last = &lastMove{From: hist[n-1].Move.From.String(), To: hist[n-1].Move.To.String()}
		}
		if pos.Sync && n > 0 && !hist[n-2].IsPass() {
			v.Also = &lastMove{From: hist[n-2].Move.From.String(), To: hist[n-2].Move.To.String()}
		}
		if pos.InCheck() && !pos.Sync {
			v.Check = pos.KingSquare(pos.Turn).String()
		}
		return v
	}

	// Ply i was played by white when i is even (games start from the initial position).
	mover := func(i int) int { return i % 2 }

	vision := pos.Vision(engine.Color(viewer))
	v.Board = boardView(&pos, &vision)
	v.FullVision = pos.Outpost(engine.Color(viewer)) != engine.NoSquare
	for i, pm := range hist {
		if mover(i) == viewer {
			v.History[i] = strings.TrimRight(pm.SAN, "+#")
		} else {
			v.History[i] = "?"
		}
	}
	// Your own last move: the last ply, or in Sync Move your half of the last round.
	lo := n - 1
	if pos.Sync {
		lo = n - 2
	}
	for i := n - 1; i >= 0 && i >= lo; i-- {
		if mover(i) == viewer && !hist[i].IsPass() {
			v.Last = &lastMove{From: hist[i].Move.From.String(), To: hist[i].Move.To.String()}
			break
		}
	}
	// You know when you're in check, since it limits your moves; never the reverse.
	if viewer == int(pos.Turn) && pos.InCheck() && !pos.Sync {
		v.Check = pos.KingSquare(pos.Turn).String()
	}
	return v
}

// enemyReach is where each enemy piece the player can see could go, as the
// player would work it out: on a copy of the board with every square they
// can't see emptied, and without castling or en passant, which depend on
// moves they may not have seen. Hidden pieces neither block nor show up, so
// nothing here is news to the player. A nil seen means everything is visible.
func enemyReach(pos *engine.Position, us engine.Color, seen *[64]bool) map[string][]string {
	view := *pos
	view.Castling, view.EP = 0, engine.NoSquare
	if seen != nil {
		for sq := range view.Board {
			if !seen[sq] {
				view.Board[sq] = engine.NoPiece
			}
		}
	}
	reach := map[string][]string{}
	for sq, pc := range view.Board {
		if pc.IsEmpty() || pc.Color() == us {
			continue
		}
		var to []string
		for _, m := range view.PieceMoves(engine.Square(sq)) {
			if s := m.To.String(); !slices.Contains(to, s) {
				to = append(to, s)
			}
		}
		if len(to) > 0 {
			reach[engine.Square(sq).String()] = to
		}
	}
	if len(reach) == 0 {
		return nil
	}
	return reach
}

// kingDanger says whether any enemy piece attacks the player's king, seen or
// not (hidden is true when none of the attackers is seen), and which of the
// king's moves land on a square a seen piece attacks. The player is told the
// king is attacked but not from where; for the unsafe squares, pieces the fog
// hides count as blockers but never as attackers, so those give nothing away.
// A nil seen means everything is visible.
func kingDanger(pos *engine.Position, us engine.Color, seen *[64]bool) (danger string, hidden bool, unsafe []string) {
	king := pos.KingSquare(us)
	if king == engine.NoSquare {
		return "", false, nil
	}
	attacks := func(p *engine.Position, target engine.Square, seen *[64]bool) bool {
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
	if attacks(pos, king, nil) {
		danger = king.String()
		hidden = !attacks(pos, king, seen)
	}
	if pos.Turn == us || pos.Sync {
		for _, m := range pos.MovesFor(us) {
			if m.From != king {
				continue
			}
			after := pos.Apply(m)
			if attacks(&after, m.To, seen) {
				unsafe = append(unsafe, m.To.String())
			}
		}
	}
	return danger, hidden, unsafe
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
	hist, toks := g.History(), g.Tokens()
	delay := spectatorDelay
	if rules.Sync {
		delay = spectatorDelay / 2 // rounds, each two plies
	}
	past, _ := engine.NewGameFromFEN(g.StartFEN(), rules)
	for _, tok := range toks[:max(0, len(toks)-delay)] {
		past.PlayToken(tok)
	}
	shown := len(past.History())
	v := viewFor(past, side, false, true)
	v.Turn = g.Turn().String()
	v.Ply = len(hist)
	v.Behind = len(hist) - shown
	return v
}
