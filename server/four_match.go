package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chess/bots"
	"chess/four"
	"github.com/coder/websocket"
)

// Four-player chess is a mods game of its own: four seats on the cross-shaped
// board, played clockwise, last king standing wins. It has its own matches,
// table and socket, and leaves the two-player Match alone.

// fourSettings are the options a four-player game can have: the mods that make
// sense with four players, and a clock. Eating the king is always the rule.
type fourSettings struct {
	Fog          bool        `json:"fog"`
	Time         TimeControl `json:"time"`
	JumpKing     bool        `json:"jumpKing,omitempty"`
	JesterPawns  bool        `json:"jesterPawns,omitempty"`
	Hill         bool        `json:"hill,omitempty"`
	CenterVision string      `json:"centerVision,omitempty"`
	// Setup is "" for the usual start, or "backRow" or "battleship" for each
	// player arranging their own pieces before the game.
	Setup string `json:"setup,omitempty"`
	// Sync is Sync Move: everyone moves at once each round.
	Sync bool `json:"sync,omitempty"`
	// Board is "" for the usual cross, or "extended" for the 16 by 16 board
	// with four rows each and neutral corners. It has no hill.
	Board string `json:"board,omitempty"`
	// Rated is a ranked game: only the four-player queue starts one, always
	// with rankedFour's settings.
	Rated bool `json:"rated,omitempty"`
	// Demo is War Mode from the home page board: you against three copies of
	// Bot, with no clock. It lives only in memory and is never saved or ranked.
	Demo bool   `json:"demo,omitempty"`
	Bot  string `json:"bot,omitempty"`
}

// warMode is War Mode: the extended board, Battleship, Sync Move, jester
// pawns, fog of war with centre vision (any piece, as in Eat the King), half
// an hour each plus ten seconds a move.
var warMode = fourSettings{Board: "extended", Setup: "battleship", Sync: true, JesterPawns: true,
	Fog: true, CenterVision: "any", Time: TimeControl{Base: 1800, Inc: 10}}

// rankedFour is the one ranked four-player game, War Mode.
var rankedFour = func() fourSettings { s := warMode; s.Rated = true; return s }()

// demoFour is War Mode against three copies of a bot from the home page
// board: no clock, as on the one-against-one demo board.
func demoFour(bot bots.Level) fourSettings {
	if !bot.Valid() {
		bot = demoBot
	}
	s := warMode
	s.Time, s.Demo, s.Bot = TimeControl{}, true, string(bot)
	return s
}

// fourArrangeTime is how long players of a ranked game have to place their
// pieces; anyone still arranging then gets a bot's arrangement.
const fourArrangeTime = 2 * time.Minute

var fourShapes = map[string]four.Shape{"": four.Cross, "extended": four.Extended}

var fourLayouts = map[string]four.Layout{"": four.Standard, "backRow": four.BackRow, "battleship": four.Battleship}

func (s fourSettings) rules() four.Rules {
	return four.Rules{JumpKing: s.JumpKing, JesterPawns: s.JesterPawns, Hill: s.Hill,
		CenterVision: four.CenterVision(centerVisions[s.CenterVision]), Sync: s.Sync, Shape: fourShapes[s.Board]}
}

func (s fourSettings) shape() four.Shape { return fourShapes[s.Board] }

func (s fourSettings) layout() four.Layout { return fourLayouts[s.Setup] }

func (s fourSettings) valid() bool {
	_, center := centerVisions[s.CenterVision]
	_, layout := fourLayouts[s.Setup]
	_, shape := fourShapes[s.Board]
	return center && layout && shape && !(s.Hill && s.Board != "") && (s.Time.valid() || s.Time.modsValid())
}

// fourProtocol changes whenever the four-player state message changes shape.
// Must match FOUR_PROTOCOL in web/src/useFourMatch.ts.
const fourProtocol = 4

// fourBotName is who fills an empty seat.
const fourBotName = "Rex inutilis"

// fourSpectatorDelay is how many saved events spectators of a fog game are
// kept behind: two rounds of four moves, or two Sync Move rounds.
const fourSpectatorDelay, fourSyncSpectatorDelay = 8, 2

// FourHub holds the four-player matches in play, and saves them.
type FourHub struct {
	mu      sync.Mutex
	matches map[string]*FourMatch
	db      *sql.DB // nil keeps games only in memory, as in tests
	store   *Store  // nil in tests: then only what everyone has is unlocked
	writes  chan fourRecord
	// botPause is how long a bot waits before moving, so its moves can be
	// seen one by one. Tests set it to nothing.
	botPause func() time.Duration
}

func NewFourHub(store *Store) *FourHub {
	h := &FourHub{matches: map[string]*FourMatch{}, writes: make(chan fourRecord, 1024),
		botPause: func() time.Duration { return time.Duration(350+rand.IntN(500)) * time.Millisecond }}
	if store != nil {
		h.db, h.store = store.db, store
	}
	go h.writer()
	go h.sweep()
	return h
}

func (h *FourHub) Create(creator *player, settings fourSettings) *FourMatch {
	h.mu.Lock()
	defer h.mu.Unlock()
	for {
		id := newID(6)
		if _, taken := h.matches[id]; taken {
			continue
		}
		if rec, _ := h.load(id); rec != nil {
			continue
		}
		now := time.Now()
		m := &FourMatch{hub: h, id: id, settings: settings, game: four.NewGame(settings.rules()),
			clients: map[*client]struct{}{}, created: now, touched: now}
		m.resetClock()
		m.seat(four.Red, creator)
		if settings.Demo {
			// The bots take the other seats and place their pieces now; the
			// game begins once you've placed yours.
			bot := bots.Level(settings.Bot)
			rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
			for _, c := range four.Colors[1:] {
				m.seat(c, &player{key: "bot:" + string(bot) + ":" + c.String(), name: bot.Name()})
				m.start.Setups[c] = four.BotSetup(settings.shape(), rng)
			}
		}
		m.save()
		h.matches[id] = m
		return m
	}
}

// CreateRanked starts a ranked game for four players from the queue, seated
// in the order given, with their four-player ratings as they start.
func (h *FourHub) CreateRanked(ps [4]*player, ratings [4]int) *FourMatch {
	h.mu.Lock()
	defer h.mu.Unlock()
	for {
		id := newID(6)
		if _, taken := h.matches[id]; taken {
			continue
		}
		if rec, _ := h.load(id); rec != nil {
			continue
		}
		now := time.Now()
		m := &FourMatch{hub: h, id: id, settings: rankedFour, game: four.NewGame(rankedFour.rules()),
			clients: map[*client]struct{}{}, created: now, touched: now, ratings: ratings}
		m.resetClock()
		for c, p := range ps {
			m.seat(four.Color(c), p)
		}
		m.armArrange()
		m.save()
		h.matches[id] = m
		return m
	}
}

// Get returns a match from memory or the database; nil if there's none.
func (h *FourHub) Get(id string) *FourMatch {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m := h.matches[id]; m != nil {
		return m
	}
	rec, err := h.load(id)
	if err != nil || rec == nil {
		return nil
	}
	m, err := restoreFour(h, rec)
	if err != nil {
		log.Printf("restoring four-player match %s: %v", id, err)
		return nil
	}
	h.matches[id] = m
	return m
}

// sweep drops matches nobody has had open for a while; they load again on demand.
func (h *FourHub) sweep() {
	for range time.Tick(5 * time.Minute) {
		h.mu.Lock()
		for id, m := range h.matches {
			m.mu.Lock()
			idle := len(m.clients) == 0 && time.Since(m.touched) > 2*time.Hour && h.db != nil
			// A demo game isn't saved, and nobody comes back to one.
			idle = idle || (m.settings.Demo && len(m.clients) == 0 && time.Since(m.touched) > 10*time.Minute)
			if idle {
				m.stopTimer()
			}
			m.mu.Unlock()
			if idle {
				delete(h.matches, id)
			}
		}
		h.mu.Unlock()
	}
}

// fourRecord is a match as saved.
type fourRecord struct {
	ID        string
	Settings  fourSettings
	Seats     [4]string
	Users     [4]int64
	Names     [4]string
	Moves     string
	Result    string
	Reason    string
	Start     fourStart
	Clock     [4]time.Duration
	TurnStart time.Time
	Created   time.Time
	Updated   time.Time
	// RatingDiff is how a ranked game moved each seat's rating, once rated.
	RatingDiff *[4]int
	// done, when set, makes this record a marker that is closed once every
	// write before it is done; see flush.
	done chan struct{}
}

// fourStart is the players' own arrangements in a game with a setup.
type fourStart struct {
	// Setups are the arrangements placed so far, "" for a player still arranging.
	Setups [4]string `json:"setups"`
	// Quit marks the players who resigned while arranging; they leave as the game starts.
	Quit [4]bool `json:"quit"`
}

// flush waits until every queued write is in the database.
func (h *FourHub) flush() {
	done := make(chan struct{})
	h.writes <- fourRecord{done: done}
	<-done
}

func (h *FourHub) writer() {
	for r := range h.writes {
		if r.done != nil {
			close(r.done)
			continue
		}
		if h.db == nil {
			continue
		}
		settings, _ := json.Marshal(r.Settings)
		seats, _ := json.Marshal(r.Seats)
		names, _ := json.Marshal(r.Names)
		users, _ := json.Marshal(r.Users)
		var ms [4]int64
		for i, d := range r.Clock {
			ms[i] = d.Milliseconds()
		}
		clock, _ := json.Marshal(ms)
		start, _ := json.Marshal(r.Start)
		var turnStart any
		if !r.TurnStart.IsZero() {
			turnStart = r.TurnStart
		}
		_, err := h.db.Exec(`
			INSERT INTO four_games (match_id, settings, seats, users, names, moves, result, reason, clock_ms, turn_started_at, created_at, updated_at, start)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (match_id) DO UPDATE SET
				seats = excluded.seats, users = excluded.users, names = excluded.names, moves = excluded.moves,
				result = excluded.result, reason = excluded.reason, clock_ms = excluded.clock_ms,
				turn_started_at = excluded.turn_started_at, updated_at = excluded.updated_at, start = excluded.start`,
			r.ID, string(settings), string(seats), string(users), string(names), r.Moves, r.Result, r.Reason,
			string(clock), turnStart, r.Created, r.Updated, string(start))
		if err != nil {
			log.Printf("saving four-player game %s: %v", r.ID, err)
		}
	}
}

// load reads a saved match; nil if there's none.
func (h *FourHub) load(id string) (*fourRecord, error) {
	if h.db == nil {
		return nil, nil
	}
	var r fourRecord
	var settings, seats, names, users, clock, start string
	var turnStart sql.NullTime
	var diff sql.NullString
	err := h.db.QueryRow(`SELECT match_id, settings, seats, users, names, moves, result, reason, clock_ms, turn_started_at, created_at, updated_at, start, rating_diff
		FROM four_games WHERE match_id = $1`, id).
		Scan(&r.ID, &settings, &seats, &users, &names, &r.Moves, &r.Result, &r.Reason, &clock, &turnStart, &r.Created, &r.Updated, &start, &diff)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settings), &r.Settings)
	json.Unmarshal([]byte(seats), &r.Seats)
	json.Unmarshal([]byte(names), &r.Names)
	json.Unmarshal([]byte(users), &r.Users)
	json.Unmarshal([]byte(start), &r.Start)
	if diff.Valid {
		r.RatingDiff = new([4]int)
		json.Unmarshal([]byte(diff.String), r.RatingDiff)
	}
	var ms [4]int64
	json.Unmarshal([]byte(clock), &ms)
	for i, n := range ms {
		r.Clock[i] = time.Duration(n) * time.Millisecond
	}
	if turnStart.Valid {
		r.TurnStart = turnStart.Time
	}
	return &r, nil
}

func restoreFour(h *FourHub, r *fourRecord) (*FourMatch, error) {
	m := &FourMatch{hub: h, id: r.ID, settings: r.Settings, game: four.NewGame(r.Settings.rules()),
		clients: map[*client]struct{}{}, created: r.Created, touched: time.Now(),
		seats: r.Seats, users: r.Users, names: r.Names, clock: r.Clock, turnStart: r.TurnStart, start: r.Start,
		ratingDiff: r.RatingDiff}
	for i, key := range r.Seats {
		m.bots[i] = strings.HasPrefix(key, "bot:")
	}
	if m.placed() {
		g, err := four.NewGameFromSetups(r.Settings.rules(), r.Start.Setups)
		if err != nil {
			return nil, err
		}
		m.game = g
	}
	for _, tok := range strings.Fields(r.Moves) {
		if err := m.game.PlayToken(tok); err != nil {
			return nil, err
		}
	}
	if !m.game.Over() && r.Result == "draw" {
		m.game.AgreeDraw()
	}
	if m.game.Over() {
		m.turnStart = time.Time{}
	}
	if m.settings.Rated {
		m.ratings = h.fourRatings(m.users)
		m.mu.Lock()
		m.armArrange()
		m.rate()
		m.mu.Unlock()
	}
	if m.running() {
		m.mu.Lock()
		if !m.flagged(time.Now()) {
			m.armTimer()
		}
		m.mu.Unlock()
	}
	m.mu.Lock()
	m.pokeBot()
	m.mu.Unlock()
	return m, nil
}

// FourMatch is one four-player game.
type FourMatch struct {
	mu       sync.Mutex
	hub      *FourHub
	id       string
	settings fourSettings
	game     *four.Game
	seats    [4]string // player keys, Red Blue Green Violet; "" while free
	users    [4]int64
	names    [4]string
	bots     [4]bool
	clients  map[*client]struct{}
	created  time.Time
	touched  time.Time
	// clock is each seat's time left as of turnStart, when the player to
	// move's clock started; a zero turnStart means no clock is running.
	clock       [4]time.Duration
	turnStart   time.Time
	timer       *time.Timer
	botThinking bool
	// start is the arrangements in a game with a setup.
	start fourStart
	// pending is each player's move for the Sync Move round in play, "" until sent.
	pending [4]string
	chat    []chatLine
	chatID  int
	// skins is how each seat's player likes their pieces drawn, as their
	// browser last said, and half which of their two sets they play in here,
	// "w" or "b", picked at random the first time. Kept only in memory.
	skins [4]*skin
	half  [4]string
	// ratings are the players' four-player ratings in a ranked game, and
	// ratingDiff how the game moved them, once it's over and rated.
	ratings    [4]int
	ratingDiff *[4]int
	rating     bool // a rating update is under way
	arrange    *time.Timer
}

func (m *FourMatch) seat(c four.Color, p *player) {
	m.seats[c], m.users[c], m.names[c] = p.key, p.userID, p.name
	m.bots[c] = strings.HasPrefix(p.key, "bot:")
}

// full is whether all four seats are taken, which is when the game starts.
func (m *FourMatch) full() bool { return !slices.Contains(m.seats[:], "") }

func (m *FourMatch) colorOf(key string) int {
	for i, s := range m.seats {
		if s != "" && s == key {
			return i
		}
	}
	return -1
}

func (m *FourMatch) online(c int) bool {
	if m.bots[c] {
		return true
	}
	for cl := range m.clients {
		if cl.playerID == m.seats[c] {
			return true
		}
	}
	return false
}

// save queues the match for the database, and rates a ranked game that has
// just ended. Callers hold m.mu.
func (m *FourMatch) save() {
	if m.settings.Demo {
		return
	}
	m.hub.writes <- fourRecord{ID: m.id, Settings: m.settings, Seats: m.seats, Users: m.users, Names: m.names,
		Moves: m.game.Tokens(), Result: m.game.Result(), Reason: m.game.Method().String(), Start: m.start,
		Clock: m.clock, TurnStart: m.turnStart, Created: m.created, Updated: time.Now()}
	m.rate()
}

// rate updates the players' four-player ratings once a ranked game is over,
// after the result is saved. Callers hold m.mu.
func (m *FourMatch) rate() {
	if !m.settings.Rated || !m.game.Over() || m.ratingDiff != nil || m.rating || m.hub.db == nil {
		return
	}
	m.rating = true
	users, places := m.users, m.game.Places()
	go func() {
		m.hub.flush()
		diff, err := m.hub.rateFour(m.id, users, places)
		m.mu.Lock()
		defer m.mu.Unlock()
		m.rating = false
		if err != nil {
			log.Printf("rating four-player game %s: %v", m.id, err)
			return
		}
		if diff != nil {
			m.ratingDiff = diff
			m.broadcast()
		}
	}()
}

// armArrange gives the players of a ranked game fourArrangeTime from the
// start to place their pieces. Callers hold m.mu.
func (m *FourMatch) armArrange() {
	if !m.settings.Rated || !m.arranging() || m.arrange != nil {
		return
	}
	m.arrange = time.AfterFunc(time.Until(m.created.Add(fourArrangeTime)), func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if !m.arranging() {
			return
		}
		rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		for c, s := range m.start.Setups {
			if s == "" {
				m.start.Setups[c] = four.BotSetup(m.settings.shape(), rng)
			}
		}
		m.begin(time.Now())
		m.save()
		m.broadcast()
	})
}

func (m *FourMatch) resetClock() {
	t := m.settings.Time
	start := time.Duration(t.Base+t.PerMove) * time.Second
	m.clock = [4]time.Duration{start, start, start, start}
	m.turnStart = time.Time{}
	m.stopTimer()
}

func (m *FourMatch) running() bool { return !m.turnStart.IsZero() }

// ticking is whether c's clock is running: the player to move's, or in a
// Sync Move round everyone's who can move and hasn't sent a move yet.
func (m *FourMatch) ticking(c int) bool {
	if !m.running() {
		return false
	}
	if !m.settings.Sync {
		return int(m.game.Turn()) == c
	}
	pos := m.game.Position()
	return !pos.Out[c] && m.pending[c] == "" && len(pos.MovesFor(four.Color(c))) > 0
}

func (m *FourMatch) left(c int, now time.Time) time.Duration {
	d := m.clock[c]
	if m.ticking(c) {
		d -= now.Sub(m.turnStart)
	}
	return d
}

// armTimer wakes the match when the first running clock runs out.
func (m *FourMatch) armTimer() {
	m.stopTimer()
	now, next := time.Now(), time.Duration(-1)
	for c := range 4 {
		if l := max(m.left(c, now), 0); m.ticking(c) && (next < 0 || l < next) {
			next = l
		}
	}
	if next >= 0 {
		m.timer = time.AfterFunc(next, m.checkFlag)
	}
}

func (m *FourMatch) stopTimer() {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// startTurn starts a turn, or a Sync Move round, and the clocks of whoever
// is to move, if the game has a clock and is on. Callers hold m.mu.
func (m *FourMatch) startTurn(now time.Time) {
	m.turnStart = time.Time{}
	m.stopTimer()
	m.pending = [4]string{}
	if !m.settings.Time.clocked() || !m.full() || m.arranging() || m.game.Over() {
		return
	}
	if pm := m.settings.Time.PerMove; pm > 0 {
		if m.settings.Sync {
			for c := range m.clock {
				m.clock[c] = time.Duration(pm) * time.Second
			}
		} else {
			m.clock[m.game.Turn()] = time.Duration(pm) * time.Second
		}
	}
	m.turnStart = now
	m.armTimer()
}

func (m *FourMatch) checkFlag() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flagged(time.Now()) {
		m.save()
		m.pokeBot()
		m.broadcast()
	} else if m.running() {
		m.armTimer()
	}
}

// flagged takes the players whose time is up out of the game, and reports
// whether there were any. Callers hold m.mu.
func (m *FourMatch) flagged(now time.Time) bool {
	if m.game.Over() || !m.running() {
		return false
	}
	var out []int
	for c := range 4 {
		if m.ticking(c) && m.left(c, now) <= 0 {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return false
	}
	for _, c := range out {
		m.clock[c] = 0
		m.game.Leave(four.Color(c), four.Timeout)
	}
	if m.settings.Sync {
		m.afterLeave(now)
	} else {
		m.startTurn(now)
	}
	return true
}

// arranging is whether the players are placing their pieces: a game with a
// setup, every seat taken, and not everyone placed yet. No clock runs.
func (m *FourMatch) arranging() bool {
	return m.settings.layout() != four.Standard && m.full() && !m.placed()
}

// placed is whether every player has placed their pieces.
func (m *FourMatch) placed() bool { return !slices.Contains(m.start.Setups[:], "") }

// begin starts a game with a setup once everyone has placed. Callers hold m.mu.
func (m *FourMatch) begin(now time.Time) {
	g, err := four.NewGameFromSetups(m.settings.rules(), m.start.Setups)
	if err != nil {
		log.Printf("starting four-player match %s: %v", m.id, err)
		return
	}
	m.game = g
	for c, quit := range m.start.Quit {
		if quit {
			g.Leave(four.Color(c), four.Resignation)
		}
	}
	m.startTurn(now)
}

// freshGame is the game as it was before the first move.
func (m *FourMatch) freshGame() *four.Game {
	if m.placed() {
		if g, err := four.NewGameFromSetups(m.settings.rules(), m.start.Setups); err == nil {
			return g
		}
	}
	return four.NewGame(m.settings.rules())
}

// submit sends c's move for the Sync Move round. A sent move is locked in.
// Callers hold m.mu.
func (m *FourMatch) submit(c int, uci string, now time.Time) bool {
	pos := m.game.Position()
	if pos.Out[c] || m.pending[c] != "" {
		return false
	}
	if _, ok := pos.FindMoveFor(four.Color(c), uci); !ok {
		return false
	}
	if m.running() {
		m.clock[c] = max(m.left(c, now), 0)
	}
	m.pending[c] = uci
	if !m.resolve(now) && m.running() {
		m.armTimer()
	}
	return true
}

// resolve plays the Sync Move round once everyone who can move has sent a
// move, and reports whether it did. Callers hold m.mu.
func (m *FourMatch) resolve(now time.Time) bool {
	if !m.settings.Sync || !m.full() || m.arranging() || m.game.Over() {
		return false
	}
	pos := m.game.Position()
	for _, c := range four.Colors {
		if !pos.Out[c] && m.pending[c] == "" && len(pos.MovesFor(c)) > 0 {
			return false
		}
	}
	if _, err := m.game.PlayRound(m.pending); err != nil {
		log.Printf("four-player match %s round %v: %v", m.id, m.pending, err)
		m.pending = [4]string{}
		return false
	}
	if m.running() {
		for c, uci := range m.pending {
			if uci != "" {
				m.clock[c] += time.Duration(m.settings.Time.Inc) * time.Second
			}
		}
	}
	m.startTurn(now)
	return true
}

// afterLeave carries on a Sync Move round after someone left it: a move that
// went for the leaver's pieces may no longer be possible, and has to be sent
// again. Callers hold m.mu.
func (m *FourMatch) afterLeave(now time.Time) {
	if m.game.Over() {
		m.startTurn(now)
		return
	}
	pos := m.game.Position()
	for c, uci := range m.pending {
		if uci == "" {
			continue
		}
		if _, ok := pos.FindMoveFor(four.Color(c), uci); pos.Out[c] || !ok {
			m.pending[c] = ""
			if m.running() && !pos.Out[c] {
				m.clock[c] += now.Sub(m.turnStart)
			}
		}
	}
	if !m.resolve(now) && m.running() {
		m.armTimer()
	}
}

// join seats the player in the first free seat while the game hasn't
// started; anyone else watches.
func (m *FourMatch) join(c *client, p *player) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.colorOf(p.key) < 0 {
		if i := m.colorOf(p.guest); i >= 0 && p.userID != 0 {
			m.seat(four.Color(i), p)
			m.save()
		} else if i := slices.Index(m.seats[:], ""); i >= 0 {
			m.seat(four.Color(i), p)
			if m.full() {
				m.startTurn(time.Now())
			}
			m.save()
		}
	}
	m.clients[c] = struct{}{}
	m.touched = time.Now()
	if color := m.colorOf(c.playerID); color >= 0 {
		c.send(m.chatJSON(true, m.chatFor(color)))
	}
	m.pokeBot()
	m.broadcast()
}

func (m *FourMatch) leave(c *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
	m.touched = time.Now()
	m.broadcast()
}

type fourIn struct {
	Type string  `json:"type"`
	UCI  string  `json:"uci"`
	Text string  `json:"text"`
	Side string  `json:"side"`
	T    float64 `json:"t"`
	// Board is an arrangement for "setup"; see four.SetupLen. "" takes it back.
	Board string `json:"board"`
	// Skin is the player's own look, sent on joining and whenever they change it.
	Skin *skin `json:"skin,omitempty"`
}

func (m *FourMatch) handle(c *client, msg fourIn) {
	if msg.Type == "skin" && !m.hub.store.stripLocked(c.playerID, msg.Skin) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	color := m.colorOf(c.playerID)
	if color < 0 {
		if side, ok := four.ParseColor(msg.Side); msg.Type == "watch" && ok && c.side != int(side) {
			c.side = int(side)
			c.push(m.state(c))
		}
		return
	}
	if msg.Type == "skin" {
		if s := cleanSkin(msg.Skin); s != m.skins[color] && (s == nil || m.skins[color] == nil || *s != *m.skins[color]) {
			m.skins[color] = s
			if m.half[color] == "" {
				m.half[color] = []string{"w", "b"}[rand.IntN(2)]
			}
			m.broadcast()
		}
		return
	}
	now := time.Now()
	m.touched = now
	events := len(m.game.History())
	switch msg.Type {
	case "fill":
		// Bots take the empty seats, and the game starts.
		if m.full() {
			return
		}
		rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		for i, s := range m.seats {
			if s == "" {
				m.seat(four.Color(i), &player{key: "bot:four:" + four.Color(i).String(), name: fourBotName})
				if m.settings.layout() != four.Standard {
					m.start.Setups[i] = four.BotSetup(m.settings.shape(), rng)
				}
			}
		}
		if m.settings.layout() != four.Standard && m.placed() {
			m.begin(now)
		} else {
			m.startTurn(now)
		}
		m.save()
	case "setup":
		if !m.arranging() || m.start.Quit[color] {
			break
		}
		if msg.Board != "" && four.CheckSetup(m.settings.shape(), m.settings.layout(), msg.Board) != nil {
			break
		}
		m.start.Setups[color] = msg.Board
		if m.placed() {
			m.begin(now)
		}
		m.save()
	case "move":
		if !m.full() || m.arranging() || m.game.Over() || m.flagged(now) {
			break
		}
		if m.settings.Sync {
			m.submit(color, msg.UCI, now)
		} else if int(m.game.Turn()) == color {
			m.play(msg.UCI, now)
		}
	case "resign":
		if !m.full() || m.game.Over() {
			break
		}
		if m.arranging() {
			// They leave as the game starts, with the usual arrangement in place of theirs.
			if m.start.Quit[color] {
				break
			}
			m.start.Quit[color] = true
			if m.start.Setups[color] == "" {
				m.start.Setups[color] = m.settings.shape().StandardSetup()
			}
			if m.placed() {
				m.begin(now)
			}
			m.save()
			break
		}
		if m.game.Position().Out[color] {
			break
		}
		if m.ticking(color) {
			m.clock[color] = max(m.left(color, now), 0)
		}
		onTurn := int(m.game.Turn()) == color
		m.game.Leave(four.Color(color), four.Resignation)
		if m.settings.Sync {
			m.afterLeave(now)
		} else if onTurn || m.game.Over() {
			m.startTurn(now)
		}
	case "chat":
		text := strings.TrimSpace(msg.Text)
		if text == "" || !utf8.ValidString(text) {
			return
		}
		if utf8.RuneCountInString(text) > chatMaxLen {
			text = string([]rune(text)[:chatMaxLen])
		}
		m.chatID++
		line := chatLine{ID: m.chatID, From: four.Color(color).String(), Text: text, At: now.UnixMilli(),
			outOnly: m.knockedOut(color)}
		m.chat = append(m.chat, line)
		if len(m.chat) > chatMaxLines {
			m.chat = slices.Clone(m.chat[len(m.chat)-chatMaxLines:])
		}
		b := m.chatJSON(false, []chatLine{line})
		for cl := range m.clients {
			if to := m.colorOf(cl.playerID); to >= 0 && (!line.outOnly || m.knockedOut(to)) {
				cl.send(b)
			}
		}
		return
	}
	if len(m.game.History()) != events {
		m.save()
	}
	m.pokeBot()
	m.broadcast()
}

// play plays the move of the player to move and runs the clocks. Callers hold m.mu.
func (m *FourMatch) play(uci string, now time.Time) bool {
	c := int(m.game.Turn())
	left := m.left(c, now)
	if _, err := m.game.PlayUCI(uci); err != nil {
		return false
	}
	if m.running() {
		m.clock[c] = left + time.Duration(m.settings.Time.Inc)*time.Second
	}
	m.startTurn(now)
	return true
}

// botsToMove are the bots that still have to send a move in a Sync Move
// round, or the bot to move. Callers hold m.mu.
func (m *FourMatch) botsToMove() []four.Color {
	if !m.full() || m.arranging() || m.game.Over() {
		return nil
	}
	if !m.settings.Sync {
		if m.bots[m.game.Turn()] {
			return []four.Color{m.game.Turn()}
		}
		return nil
	}
	var out []four.Color
	pos := m.game.Position()
	for _, c := range four.Colors {
		if m.bots[c] && !pos.Out[c] && m.pending[c] == "" && len(pos.MovesFor(c)) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// pokeBot has the bots move after a short pause when it's their turn, or in
// a Sync Move round. Callers hold m.mu.
func (m *FourMatch) pokeBot() {
	if m.botThinking || len(m.botsToMove()) == 0 {
		return
	}
	m.botThinking = true
	events := len(m.game.History())
	go func() {
		time.Sleep(m.hub.botPause())
		m.mu.Lock()
		defer m.mu.Unlock()
		m.botThinking = false
		bots := m.botsToMove()
		if len(m.game.History()) != events || len(bots) == 0 {
			m.pokeBot()
			return
		}
		rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		now := time.Now()
		var moves []string
		for _, c := range bots {
			pos := m.game.Position()
			pos.Turn = c
			mv, _ := fourBotMove(m.seats[c], &pos, m.settings.Fog, rng)
			moves = append(moves, mv.UCI())
		}
		for i, c := range bots {
			if m.settings.Sync {
				m.submit(int(c), moves[i], now)
			} else {
				m.play(moves[i], now)
			}
		}
		if len(m.game.History()) != events {
			m.save()
		}
		m.pokeBot()
		m.broadcast()
	}()
}

// fourBotMove is the move of the bot in a seat. Rex inutilis plays at random,
// as in one-against-one; the others play four.BotMoveSeeing at their level,
// from what they can see when the game has fog.
func fourBotMove(seat string, pos *four.Position, fog bool, rng *rand.Rand) (four.Move, bool) {
	level := bots.Level(strings.Split(strings.TrimPrefix(seat, "bot:"), ":")[0])
	if level == bots.Rex {
		moves := pos.LegalMoves()
		if len(moves) == 0 {
			return four.Move{}, false
		}
		return moves[rng.IntN(len(moves))], true
	}
	var seen *[four.NumSquares]bool
	if fog {
		v := pos.Vision(pos.Turn)
		seen = &v
	}
	return four.BotMoveSeeing(pos, seen, fourStrength[level], rng)
}

// fourStrength is how each bot plays four-player chess; the seats the queue
// fills ("bot:four:<seat>") play the strongest.
var fourStrength = map[bots.Level]four.Strength{
	bots.Vercingetorix: four.Greedy,
	bots.Ramesses:      four.Careful,
	bots.Solomon:       four.Strongest,
	"four":             four.Strongest,
}

func (m *FourMatch) chatJSON(reset bool, lines []chatLine) []byte {
	if lines == nil {
		lines = []chatLine{}
	}
	b, _ := json.Marshal(chatMsg{Type: "chat", Reset: reset, Lines: lines})
	return b
}

type fourSeat struct {
	Taken  bool   `json:"taken"`
	Online bool   `json:"online"`
	Name   string `json:"name,omitempty"`
	Bot    bool   `json:"bot,omitempty"`
	// Out is how the player left the game: "capturing the king", "resignation" or "time".
	Out string `json:"out,omitempty"`
	// Ready is set once the player has sent a move for the Sync Move round,
	// or placed their pieces in a game with a setup.
	Ready bool `json:"ready,omitempty"`
	// Skin is the player's own look, and Half the one of their White and
	// Black sets they play in this game ("w" or "b"). A seat without one
	// plays in its own colour.
	Skin *skin  `json:"skin,omitempty"`
	Half string `json:"half,omitempty"`
	// Rating is the player's four-player rating in a ranked game, and
	// RatingDiff how the game moved it, once rated.
	Rating     int  `json:"rating,omitempty"`
	RatingDiff *int `json:"ratingDiff,omitempty"`
}

type fourEvent struct {
	By  string `json:"by"`
	SAN string `json:"san"` // "?" for a move the viewer can't know
	Out string `json:"out,omitempty"`
	// Round is the Sync Move round the move was part of.
	Round int `json:"round,omitempty"`
}

type fourView struct {
	// Board is every square from the bottom left, row by row, width by width
	// (14 or 16): " " off the board, "." empty, "?" hidden, or the piece's
	// colour and kind, like "rK".
	Board []string  `json:"board"`
	Turn  string    `json:"turn"`
	Ply   int       `json:"ply"`
	Legal []string  `json:"legal"`
	Last  *lastMove `json:"last"`
	// LastRound is every move of the last Sync Move round, when the viewer can see them all.
	LastRound    []lastMove `json:"lastRound,omitempty"`
	Danger       string     `json:"danger,omitempty"`
	DangerHidden bool       `json:"dangerHidden,omitempty"`
	// Reach is where each other player's piece the viewer can see could move,
	// worked out on the board as the viewer sees it (see enemyReach in view.go).
	Reach      map[string][]string `json:"reach,omitempty"`
	FullVision bool                `json:"fullVision,omitempty"`
	History    []fourEvent         `json:"history"`
	Captured   map[string][]string `json:"captured"`
	Behind     int                 `json:"behind,omitempty"`
}

type fourState struct {
	Type     string       `json:"type"`
	Protocol int          `json:"protocol"`
	ID       string       `json:"id"`
	You      string       `json:"you"`
	Settings fourSettings `json:"settings"`
	fourView
	Status     string              `json:"status"` // waiting, setup, playing, over
	Result     string              `json:"result,omitempty"`
	Reason     string              `json:"reason,omitempty"`
	Players    map[string]fourSeat `json:"players"`
	Clock      *clockView          `json:"clock,omitempty"`
	Watching   string              `json:"watching,omitempty"`
	Spectators int                 `json:"spectators"`
	// Ticking is whose clocks run in a Sync Move round.
	Ticking []string `json:"ticking,omitempty"`
	// Planned is the move the viewer has sent for this Sync Move round.
	Planned string `json:"planned,omitempty"`
	// Arranged is the viewer's arrangement while players place their pieces.
	Arranged string `json:"arranged,omitempty"`
}

// fourReach is enemyReach for four players: where each other player's piece
// that us can see could go, on a copy of the board with every square us can't
// see emptied and no castling or en passant.
func fourReach(pos *four.Position, us four.Color, seen *[four.NumSquares]bool) map[string][]string {
	view := *pos
	view.Castling, view.EP, view.EPPawn = 0, four.NoSquare, four.NoSquare
	if seen != nil {
		for sq := range view.Board {
			if !seen[sq] {
				view.Board[sq] = four.NoPiece
			}
		}
	}
	reach := map[string][]string{}
	for sq, pc := range view.Board {
		if pc.IsEmpty() || pc.Color() == us || view.Out[pc.Color()] {
			continue
		}
		var to []string
		for _, m := range view.PieceMoves(four.Square(sq)) {
			if s := m.To.String(); !slices.Contains(to, s) {
				to = append(to, s)
			}
		}
		if len(to) > 0 {
			reach[four.Square(sq).String()] = to
		}
	}
	if len(reach) == 0 {
		return nil
	}
	return reach
}

// fourViewFor is the game as viewer sees it (-1 sees everything). Under fog a
// player sees what their pieces see, their own moves, and who is out, but not
// the other moves; a player who is out sees everything.
func fourViewFor(g *four.Game, viewer int, playable, fog bool) fourView {
	pos := g.Position()
	hist := g.History()
	w := pos.Shape.Width()
	v := fourView{Turn: pos.Turn.String(), Ply: len(hist), Legal: []string{}, History: make([]fourEvent, len(hist)),
		Captured: map[string][]string{}, Board: make([]string, w*w)}
	hidden := fog && !g.Over() && viewer >= 0 && !pos.Out[viewer]
	var seen *[four.NumSquares]bool
	if hidden {
		vision := pos.Vision(four.Color(viewer))
		seen = &vision
		v.FullVision = pos.Outpost(four.Color(viewer)) != four.NoSquare
	}
	for i := range v.Board {
		sq := four.At(i%w, i/w)
		pc := pos.Board[sq]
		switch {
		case !pos.Shape.Valid(sq):
			v.Board[i] = " "
		case seen != nil && !seen[sq]:
			v.Board[i] = "?"
		case pc.IsEmpty():
			v.Board[i] = "."
		default:
			v.Board[i] = pc.Code()
		}
	}
	if playable && !g.Over() && viewer >= 0 && !pos.Out[viewer] && (pos.Sync || viewer == int(pos.Turn)) {
		for _, mv := range pos.MovesFor(four.Color(viewer)) {
			v.Legal = append(v.Legal, mv.UCI())
		}
	}
	if viewer >= 0 && !g.Over() && !pos.Out[viewer] {
		if k := pos.KingSquare(four.Color(viewer)); k != four.NoSquare && pos.Attacked(k, four.Color(viewer), nil) {
			v.Danger = k.String()
			v.DangerHidden = seen != nil && !pos.Attacked(k, four.Color(viewer), seen)
		}
		if playable {
			v.Reach = fourReach(&pos, four.Color(viewer), seen)
		}
	}
	for _, c := range four.Colors {
		v.Captured[c.String()] = []string{}
	}
	for i, e := range hist {
		mine := !hidden || int(e.By) == viewer
		v.History[i] = fourEvent{By: e.By.String(), SAN: "?", Round: e.Round}
		if e.Out != four.NoMethod {
			v.History[i] = fourEvent{By: e.By.String(), Out: e.Out.String()}
			continue
		}
		if mine {
			v.History[i].SAN = e.SAN
			if !e.Captured.IsEmpty() {
				v.Captured[e.By.String()] = append(v.Captured[e.By.String()], e.Captured.Code())
			}
		}
	}
	// The last move the viewer knows: the last one played, or under fog their own.
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Out == four.NoMethod && (!hidden || int(hist[i].By) == viewer) {
			v.Last = &lastMove{From: hist[i].Move.From.String(), To: hist[i].Move.To.String()}
			break
		}
	}
	if n := len(hist); n > 0 && !hidden && hist[n-1].Round > 0 {
		for i := n - 1; i >= 0 && hist[i].Round == hist[n-1].Round; i-- {
			v.LastRound = append(v.LastRound, lastMove{From: hist[i].Move.From.String(), To: hist[i].Move.To.String()})
		}
	}
	return v
}

// base is the part of the state that's the same for everyone. Callers hold m.mu.
func (m *FourMatch) base() fourState {
	s := fourState{Type: "state", Protocol: fourProtocol, ID: m.id, Settings: m.settings, Status: "playing",
		Players: map[string]fourSeat{}}
	switch {
	case m.game.Over():
		s.Status, s.Result, s.Reason = "over", m.game.Result(), m.game.Method().String()
	case !m.full():
		s.Status = "waiting"
	case m.arranging():
		s.Status = "setup"
	}
	for _, c := range four.Colors {
		seat := fourSeat{Taken: m.seats[c] != "", Name: m.names[c], Bot: m.bots[c], Skin: m.skins[c]}
		if seat.Skin != nil {
			seat.Half = m.half[c]
		}
		seat.Online = seat.Taken && m.online(int(c))
		if m.settings.Rated {
			seat.Rating = m.ratings[c]
			if m.ratingDiff != nil {
				seat.RatingDiff = &m.ratingDiff[c]
			}
		}
		if f := m.game.Fell(c); f != four.NoMethod {
			seat.Out = f.String()
		}
		if s.Status == "setup" {
			seat.Ready = m.start.Setups[c] != ""
			if m.start.Quit[c] {
				seat.Out = four.Resignation.String()
			}
		} else {
			seat.Ready = m.pending[c] != ""
		}
		s.Players[c.String()] = seat
	}
	if m.settings.Time.clocked() {
		now := time.Now()
		s.Clock = &clockView{Left: map[string]int64{}}
		for _, c := range four.Colors {
			s.Clock.Left[c.String()] = max(m.left(int(c), now), 0).Milliseconds()
		}
		for _, c := range four.Colors {
			if m.ticking(int(c)) {
				s.Ticking = append(s.Ticking, c.String())
			}
		}
		if m.running() && !m.settings.Sync {
			s.Clock.Running = m.game.Turn().String()
		}
	}
	seen := map[string]bool{}
	for cl := range m.clients {
		if m.colorOf(cl.playerID) < 0 {
			seen[cl.playerID] = true
		}
	}
	s.Spectators = len(seen)
	return s
}

// knockedOut reports whether a seated player is out while the game goes on.
// They know more than the players still in (they see more of the board), so
// what they write reaches only others who are out, and they watch a few
// rounds behind like spectators. Callers hold m.mu.
func (m *FourMatch) knockedOut(color int) bool {
	return color >= 0 && !m.game.Over() && m.game.Position().Out[color]
}

// chatFor is the chat a seated player may read: lines from players who were
// out while the game ran are left out for players still in it, until it ends.
// Callers hold m.mu.
func (m *FourMatch) chatFor(color int) []chatLine {
	if m.game.Over() || m.knockedOut(color) {
		return m.chat
	}
	lines := make([]chatLine, 0, len(m.chat))
	for _, l := range m.chat {
		if !l.outOnly {
			lines = append(lines, l)
		}
	}
	return lines
}

// view fills in the state for a player (color 0 to 3) or a spectator who
// watches as side. Callers hold m.mu.
func (m *FourMatch) view(s fourState, color, side int) []byte {
	if color >= 0 {
		s.You = four.Color(color).String()
		s.fourView = fourViewFor(m.game, color, s.Status == "playing", m.settings.Fog)
		if m.settings.Fog && m.knockedOut(color) {
			// Out of the game, they'd see everything live and could pass it on.
			s.fourView = m.spectatorView(color)
		}
		s.Planned = m.pending[color]
		if s.Planned != "" {
			s.Legal = []string{}
		}
		if s.Status == "setup" {
			s.Arranged = m.start.Setups[color]
		}
	} else {
		s.Watching = four.Color(side).String()
		s.fourView = m.spectatorView(side)
	}
	if s.Status == "setup" {
		// Nobody sees anybody's arrangement; the page shows the viewer's own.
		for i, sq := range s.Board {
			if sq != " " {
				s.Board[i] = "."
			}
		}
	}
	b, _ := json.Marshal(s)
	return b
}

// spectatorView keeps spectators of a fog game a couple of rounds behind, seeing
// what side saw then, so nobody watching can tell a player what's hidden.
func (m *FourMatch) spectatorView(side int) fourView {
	if !m.settings.Fog || m.game.Over() {
		return fourViewFor(m.game, -1, false, m.settings.Fog)
	}
	hist, toks := m.game.History(), m.game.TokenList()
	delay := fourSpectatorDelay
	if m.settings.Sync {
		delay = fourSyncSpectatorDelay
	}
	past := m.freshGame()
	for _, tok := range toks[:max(0, len(toks)-delay)] {
		past.PlayToken(tok)
	}
	v := fourViewFor(past, side, false, true)
	v.Turn, v.Ply, v.Behind = m.game.Turn().String(), len(hist), len(hist)-len(past.History())
	return v
}

func (m *FourMatch) state(c *client) []byte {
	return m.view(m.base(), m.colorOf(c.playerID), c.side)
}

// broadcast sends every client its own view. Callers hold m.mu.
func (m *FourMatch) broadcast() {
	base := m.base()
	views := map[int][]byte{}
	for c := range m.clients {
		slot := m.colorOf(c.playerID)
		if slot < 0 {
			slot = 4 + c.side
		}
		if views[slot] == nil {
			views[slot] = m.view(base, m.colorOf(c.playerID), c.side)
		}
		c.push(views[slot])
	}
}

// serve runs one websocket connection until it closes.
func (m *FourMatch) serve(ctx context.Context, conn *websocket.Conn, p *player) {
	c := newClient(p.key)
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
	m.join(c, p)
	defer m.leave(c)
	conn.SetReadLimit(4096)
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				conn.Close(websocket.StatusNormalClosure, "")
			}
			return
		}
		var msg fourIn
		if json.Unmarshal(b, &msg) != nil {
			continue
		}
		if msg.Type == "ping" {
			pong, _ := json.Marshal(map[string]any{"type": "pong", "t": msg.T})
			c.send(pong)
			continue
		}
		m.handle(c, msg)
	}
}
