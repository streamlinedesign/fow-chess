package main

import (
	"strings"

	"chess/engine"
)

// Settings are chosen when a match is created, for play testing different rules.
type Settings struct {
	// Fog hides every square the viewer's pieces can't see.
	Fog bool `json:"fog"`
	// KingCapture plays without check: win by capturing the king.
	KingCapture bool `json:"kingCapture"`
}

func (s Settings) rules() engine.Rules { return engine.Rules{KingCapture: s.KingCapture} }

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
	Board   string    `json:"board"`
	Turn    string    `json:"turn"`
	Ply     int       `json:"ply"`
	Legal   []string  `json:"legal"`
	Last    *lastMove `json:"last"`
	Check   string    `json:"check,omitempty"`
	History []string  `json:"history"` // SAN, or "?" for moves the viewer may not know
}

// viewFor builds the game as seen by viewer (-1 for spectators). Legal moves are
// only sent to the player whose turn it is, so the browser never has to know the rules.
//
// With fog, a player sees only what their pieces see and gets no hint of the
// opponent's moves: no last-move highlight, no notation, and no check marks on
// their own moves. Spectators see just the last move of each side. Once the
// game is over everyone sees everything.
func viewFor(g *engine.Game, viewer int, playable, fog bool) gameView {
	pos := g.Position()
	hist := g.History()
	n := len(hist)
	v := gameView{
		Turn:    pos.Turn.String(),
		Ply:     n,
		Legal:   []string{},
		History: make([]string, n),
	}
	if playable && !g.Over() && viewer == int(pos.Turn) {
		for _, m := range g.LegalMoves() {
			v.Legal = append(v.Legal, m.UCI())
		}
	}

	if !fog || g.Over() {
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

	if viewer < 0 {
		var visible [64]bool
		for i := range v.History {
			v.History[i] = "?"
		}
		for i := max(0, n-2); i < n; i++ {
			m := hist[i].Move
			visible[m.From], visible[m.To] = true, true
			v.History[i] = hist[i].SAN
		}
		v.Board = boardView(&pos, &visible)
		if n > 0 {
			v.Last = &lastMove{From: hist[n-1].Move.From.String(), To: hist[n-1].Move.To.String()}
		}
		return v
	}

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
