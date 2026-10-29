// Package sim plays Eat the King between bots, with no server, browser or
// database, so thousands of games can be played to see how the game behaves.
package sim

import (
	"math/rand/v2"

	"chess/engine"
)

// View is everything a player knows when it's their move: what a real player
// gets from the server, and nothing more. Under fog, squares the player can't
// see are empty in Board and false in Seen.
type View struct {
	Me    engine.Color
	Ply   int
	Fog   bool
	Board [64]engine.Piece
	Seen  [64]bool
	// Legal is the moves the server would offer. Every square a move lands on
	// is one the player sees, so the list leaks nothing the board doesn't.
	Legal []engine.Move
	// Castling is the player's own castling rights (the opponent's are unknown).
	Castling uint8
	Rules    engine.Rules
	// pos is the real position, for bots such as the site's own (server/bots)
	// that are handed it with the seen mask and hide the fog themselves.
	pos engine.Position
}

// Bot picks moves for one side of one game. A new Bot is made for every game,
// so a bot may remember what it saw on earlier turns.
type Bot interface {
	Move(v *View) engine.Move
}

// BotMaker makes a bot for one game, seeded so a game can be played again exactly.
type BotMaker func(rng *rand.Rand) Bot

// Bots are the players the simulator knows, by name.
var Bots = map[string]BotMaker{
	"random": func(rng *rand.Rand) Bot { return &randomBot{rng} },
	"greedy": func(rng *rand.Rand) Bot { return &greedyBot{rng: rng} },
	"hunter": func(rng *rand.Rand) Bot { return &hunter{rng: rng} },
	"wary":   func(rng *rand.Rand) Bot { return &hunter{rng: rng, wary: true} },
}

// viewOf builds what color c knows in pos.
func viewOf(g *engine.Game, c engine.Color, fog bool, rules engine.Rules) *View {
	pos := g.Position()
	v := &View{Me: c, Ply: len(g.History()), Fog: fog, Legal: g.LegalMoves(), Rules: rules, pos: pos}
	if fog {
		v.Seen = pos.Vision(c)
	} else {
		for i := range v.Seen {
			v.Seen[i] = true
		}
	}
	for i, pc := range pos.Board {
		if v.Seen[i] {
			v.Board[i] = pc
		}
	}
	mine := engine.CastleWK | engine.CastleWQ
	if c == engine.Black {
		mine = engine.CastleBK | engine.CastleBQ
	}
	v.Castling = pos.Castling & mine
	return v
}

// value is a piece's worth in pawns. The king is worth the game.
var value = [...]int{engine.NoKind: 0, engine.Pawn: 1, engine.Knight: 3, engine.Bishop: 3, engine.Rook: 5, engine.Queen: 9, engine.King: 1000}

type randomBot struct{ rng *rand.Rand }

func (b *randomBot) Move(v *View) engine.Move { return v.Legal[b.rng.IntN(len(v.Legal))] }

// greedyBot eats the king when it can see it, otherwise takes the most
// valuable piece it can without losing more, otherwise makes a move that
// no piece it can see attacks.
type greedyBot struct{ rng *rand.Rand }

func (b *greedyBot) Move(v *View) engine.Move {
	pos := beliefPosition(v, nil)
	best, bestScore := engine.Move{}, -1<<30
	for _, m := range v.Legal {
		gain := value[v.Board[m.To].Kind()] * 10
		if v.Board[m.To].Kind() == engine.King {
			return m
		}
		after := pos.Apply(m)
		if after.Attacked(m.To, v.Me.Other()) {
			gain -= value[v.Board[m.From].Kind()] * 10
		}
		if m.Promo == engine.Queen {
			gain += 80
		}
		score := gain*100 + b.rng.IntN(100)
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best
}

// beliefPosition is the board as v's player believes it: what they see, plus
// pieces they remember (ghosts), with v.Me to move.
func beliefPosition(v *View, ghosts *[64]engine.Piece) engine.Position {
	p := engine.Position{Board: v.Board, Turn: v.Me, Castling: v.Castling, EP: engine.NoSquare, Fullmove: 1}
	p.KingCapture, p.JumpKing, p.BackPawns, p.Hill = v.Rules.KingCapture, v.Rules.JumpKing, v.Rules.BackPawns, v.Rules.Hill
	if ghosts != nil {
		for i, g := range ghosts {
			if g != engine.NoPiece && !v.Seen[i] {
				p.Board[i] = g
			}
		}
	}
	return p
}
