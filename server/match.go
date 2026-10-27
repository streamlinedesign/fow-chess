package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"chess/engine"
	"github.com/coder/websocket"
)

// idAlphabet skips look-alike characters (0/O, 1/I/L) so IDs are easy to read aloud.
const idAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const (
	white = 0
	black = 1
)

var colorNames = [2]string{"w", "b"}

type Hub struct {
	mu      sync.Mutex
	matches map[string]*Match
}

func NewHub() *Hub {
	h := &Hub{matches: make(map[string]*Match)}
	go h.sweep()
	return h
}

func (h *Hub) Create(creator string, settings Settings) *Match {
	h.mu.Lock()
	defer h.mu.Unlock()
	for {
		id := newID(6)
		if _, taken := h.matches[id]; taken {
			continue
		}
		m := &Match{id: id, settings: settings, game: engine.NewGameWithRules(settings.rules()), clients: make(map[*client]struct{}), touched: time.Now()}
		m.seats[white] = creator
		h.matches[id] = m
		return m
	}
}

func (h *Hub) Get(id string) *Match {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.matches[id]
}

// sweep drops matches nobody has been connected to for a while.
func (h *Hub) sweep() {
	for range time.Tick(5 * time.Minute) {
		h.mu.Lock()
		for id, m := range h.matches {
			m.mu.Lock()
			idle := len(m.clients) == 0 && time.Since(m.touched) > 2*time.Hour
			m.mu.Unlock()
			if idle {
				delete(h.matches, id)
			}
		}
		h.mu.Unlock()
	}
}

func newID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(b)
}

type client struct {
	playerID string
	mu       sync.Mutex
	latest   []byte        // newest unsent state; older unsent states are dropped
	pongs    [][]byte      // ping replies, always sent
	wake     chan struct{} // signals the writer that something is queued
}

func newClient(playerID string) *client {
	return &client{playerID: playerID, wake: make(chan struct{}, 1)}
}

// push queues a state message. A slow client skips straight to the newest state
// instead of falling behind or missing one.
func (c *client) push(b []byte) {
	c.mu.Lock()
	c.latest = b
	c.mu.Unlock()
	c.signal()
}

func (c *client) pong(b []byte) {
	c.mu.Lock()
	c.pongs = append(c.pongs, b)
	c.mu.Unlock()
	c.signal()
}

func (c *client) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *client) drain() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.pongs
	c.pongs = nil
	if c.latest != nil {
		out = append(out, c.latest)
		c.latest = nil
	}
	return out
}

type Match struct {
	mu       sync.Mutex
	id       string
	settings Settings
	game     *engine.Game
	seats    [2]string // player IDs holding white and black
	clients  map[*client]struct{}
	rematch  [2]bool
	draw     [2]bool
	round    int
	touched  time.Time
}

type seatState struct {
	Taken  bool `json:"taken"`
	Online bool `json:"online"`
}

// protocolVersion changes whenever the state message changes shape, so a web app
// talking to a server from another version can say so instead of breaking.
const protocolVersion = 3

type stateMsg struct {
	Type     string   `json:"type"`
	Protocol int      `json:"protocol"`
	ID       string   `json:"id"`
	Round    int      `json:"round"`
	You      string   `json:"you"`
	Settings Settings `json:"settings"`
	gameView
	Status  string               `json:"status"`
	Result  string               `json:"result,omitempty"`
	Reason  string               `json:"reason,omitempty"`
	Players map[string]seatState `json:"players"`
	Rematch map[string]bool      `json:"rematch"`
	Draw    map[string]bool      `json:"draw"`
}

type inMsg struct {
	Type string  `json:"type"`
	UCI  string  `json:"uci"`
	T    float64 `json:"t"` // ping timestamp, echoed back
}

// join seats the player if a seat is free (white first) and returns their color, or -1 for spectators.
func (m *Match) join(c *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.colorOf(c.playerID) < 0 {
		for i := range m.seats {
			if m.seats[i] == "" {
				m.seats[i] = c.playerID
				break
			}
		}
	}
	m.clients[c] = struct{}{}
	m.touched = time.Now()
	m.broadcast()
}

func (m *Match) leave(c *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
	m.touched = time.Now()
	m.broadcast()
}

func (m *Match) colorOf(playerID string) int {
	for i, p := range m.seats {
		if p != "" && p == playerID {
			return i
		}
	}
	return -1
}

func (m *Match) online(color int) bool {
	for c := range m.clients {
		if c.playerID == m.seats[color] {
			return true
		}
	}
	return false
}

func (m *Match) handle(c *client, msg inMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	color := m.colorOf(c.playerID)
	if color < 0 {
		return
	}
	over := m.game.Over()
	m.touched = time.Now()

	switch msg.Type {
	case "move":
		if over || m.seats[black] == "" || int(m.game.Turn()) != color {
			break
		}
		if _, err := m.game.PlayUCI(msg.UCI); err != nil {
			break
		}
		m.draw = [2]bool{}
	case "resign":
		if !over && len(m.game.History()) > 0 {
			m.game.Resign(engine.Color(color))
		}
	case "draw":
		if over || m.seats[black] == "" {
			break
		}
		m.draw[color] = !m.draw[color]
		if m.draw[white] && m.draw[black] {
			m.game.AgreeDraw()
		}
	case "rematch":
		if !over {
			break
		}
		m.rematch[color] = !m.rematch[color]
		if m.rematch[white] && m.rematch[black] {
			m.game = engine.NewGameWithRules(m.settings.rules())
			m.seats[white], m.seats[black] = m.seats[black], m.seats[white]
			m.rematch = [2]bool{}
			m.draw = [2]bool{}
			m.round++
		}
	}
	m.broadcast()
}

// broadcast sends every client its own view of the match. Callers hold m.mu.
func (m *Match) broadcast() {
	status, result, reason := "playing", "", ""
	switch {
	case m.game.Over():
		status, result, reason = "over", m.game.Outcome().String(), m.game.Method().String()
	case m.seats[black] == "":
		status = "waiting"
	}
	base := stateMsg{
		Type: "state", Protocol: protocolVersion, ID: m.id, Round: m.round, Settings: m.settings,
		Status: status, Result: result, Reason: reason,
		Players: map[string]seatState{},
		Rematch: map[string]bool{"w": m.rematch[white], "b": m.rematch[black]},
		Draw:    map[string]bool{"w": m.draw[white], "b": m.draw[black]},
	}
	for i, name := range colorNames {
		base.Players[name] = seatState{Taken: m.seats[i] != "", Online: m.seats[i] != "" && m.online(i)}
	}
	playable := status == "playing"
	var views [3][]byte // white, black, spectator; built once per broadcast
	for c := range m.clients {
		color := m.colorOf(c.playerID)
		slot := color
		if slot < 0 {
			slot = 2
		}
		if views[slot] == nil {
			msg := base
			if color >= 0 {
				msg.You = colorNames[color]
			}
			msg.gameView = viewFor(m.game, color, playable, m.settings.Fog)
			views[slot], _ = json.Marshal(msg)
		}
		c.push(views[slot])
	}
}

// serve runs one websocket connection until it closes.
func (m *Match) serve(ctx context.Context, conn *websocket.Conn, playerID string) {
	c := newClient(playerID)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ping := time.NewTicker(25 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.wake:
				for _, b := range c.drain() {
					wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
					err := conn.Write(wctx, websocket.MessageText, b)
					wcancel()
					if err != nil {
						cancel()
						return
					}
				}
			case <-ping.C:
				if err := conn.Ping(ctx); err != nil {
					cancel()
					return
				}
			}
		}
	}()

	m.join(c)
	defer m.leave(c)

	conn.SetReadLimit(1024)
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				conn.Close(websocket.StatusNormalClosure, "")
			}
			return
		}
		var msg inMsg
		if json.Unmarshal(b, &msg) != nil {
			continue
		}
		if msg.Type == "ping" {
			pong, _ := json.Marshal(map[string]any{"type": "pong", "t": msg.T})
			c.pong(pong)
			continue
		}
		start := time.Now()
		m.handle(c, msg)
		if *verbose {
			log.Printf("match %s: %s %s handled in %s", m.id, msg.Type, msg.UCI, time.Since(start))
		}
	}
}
