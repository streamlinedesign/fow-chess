package main

import (
	"bytes"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"chess/bots"
	"chess/engine"
)

// Link previews. Every page of the app is the same index.html; before it goes
// out, the server swaps its <title> for the page's own title, description and
// OpenGraph and Twitter tags, so a link pasted into a chat unfolds into a card
// with a picture (see og.go). The picture is at /og/<page>.png: /og/home.png
// for /, /og/m/ABC123.png for /m/ABC123, /og/mods.png?<query> for a mods link.
//
// Games still being played never show their board: a preview would lift the fog.

// card is what a page's preview says.
type card struct {
	Title string
	// Lines are shown under the title in the picture, and joined into the description.
	Lines []string
	// Board is what the picture shows, as boardView writes it.
	Board string
	Hill  bool
	// Fresh is how long the card may be cached: a game's status changes, a finished game doesn't.
	Fresh time.Duration
	// NotFound pages are served as usual but say nothing about what's missing.
	NotFound bool
}

// sharePlayer is what a profile's preview needs.
type sharePlayer struct {
	Name       string
	Rating     int
	FourRating int
	Title      string
	Games      int
}

// shareSource looks things up for previews. The Store is one; tests use a fake.
type shareSource interface {
	ShareGame(id string, round int) (*gameRecord, error)
	SharePlayer(name string) (*sharePlayer, error)
}

func (s *Store) ShareGame(id string, round int) (*gameRecord, error) {
	if round <= 0 {
		return s.LatestGame(id)
	}
	return s.Game(id, round)
}

func (s *Store) SharePlayer(name string) (*sharePlayer, error) {
	u, err := s.UserByName(name)
	if err != nil || u == nil {
		return nil, err
	}
	p := &sharePlayer{Name: u.Username, Rating: int(u.Rating + 0.5), FourRating: int(u.FourRating + 0.5)}
	if t, err := s.Titles([]string{u.Username}); err == nil {
		p.Title = t[u.Username]
	}
	if r, err := s.Record(u.ID, 0); err == nil {
		p.Games = r.Wins + r.Draws + r.Losses
	}
	return p, nil
}

// Sharer serves index.html with each page's preview tags, and the pictures.
type Sharer struct {
	src shareSource
	// base is the site's address for absolute links, from BASE_URL; empty
	// means the address the request came to.
	base  string
	index []byte

	mu    sync.Mutex
	cache map[string]cachedCard
}

type cachedCard struct {
	png     []byte
	expires time.Time
}

const ogCacheSize = 500

func NewSharer(src shareSource, base string, index []byte) *Sharer {
	return &Sharer{src: src, base: strings.TrimRight(base, "/"), index: index, cache: map[string]cachedCard{}}
}

// origin is the site's address: BASE_URL, or else the one the request came to.
func (s *Sharer) origin(r *http.Request) string {
	if s.base != "" {
		return s.base
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

var titleTagRE = regexp.MustCompile(`<title>[^<]*</title>`)

// ServePage writes index.html with the preview tags for r's page.
func (s *Sharer) ServePage(w http.ResponseWriter, r *http.Request) {
	c := s.card(r.URL.Path, r.URL.Query())
	origin := s.origin(r)
	page := origin + r.URL.Path
	pic := origin + "/og" + r.URL.Path + ".png"
	if r.URL.Path == "/" {
		pic = origin + "/og/home.png"
	}
	if r.URL.Path == "/mods" && r.URL.RawQuery != "" {
		page += "?" + r.URL.RawQuery
		pic += "?" + r.URL.RawQuery
	}
	desc := strings.Join(c.Lines, " · ")
	e := html.EscapeString
	var b strings.Builder
	title := c.Title
	if title != "Eat the King" {
		title += " · Eat the King"
	}
	fmt.Fprintf(&b, "<title>%s</title>\n", e(title))
	fmt.Fprintf(&b, `    <meta name="description" content="%s" />`+"\n", e(desc))
	fmt.Fprintf(&b, `    <link rel="canonical" href="%s" />`+"\n", e(page))
	fmt.Fprintf(&b, `    <meta property="og:site_name" content="Eat the King" />`+"\n")
	fmt.Fprintf(&b, `    <meta property="og:type" content="website" />`+"\n")
	fmt.Fprintf(&b, `    <meta property="og:title" content="%s" />`+"\n", e(c.Title))
	fmt.Fprintf(&b, `    <meta property="og:description" content="%s" />`+"\n", e(desc))
	fmt.Fprintf(&b, `    <meta property="og:url" content="%s" />`+"\n", e(page))
	fmt.Fprintf(&b, `    <meta property="og:image" content="%s" />`+"\n", e(pic))
	fmt.Fprintf(&b, `    <meta property="og:image:width" content="%d" />`+"\n", ogW)
	fmt.Fprintf(&b, `    <meta property="og:image:height" content="%d" />`+"\n", ogH)
	fmt.Fprintf(&b, `    <meta property="og:image:alt" content="%s" />`+"\n", e(c.Title))
	fmt.Fprintf(&b, `    <meta name="twitter:card" content="summary_large_image" />`+"\n")
	fmt.Fprintf(&b, `    <meta name="twitter:title" content="%s" />`+"\n", e(c.Title))
	fmt.Fprintf(&b, `    <meta name="twitter:description" content="%s" />`+"\n", e(desc))
	fmt.Fprintf(&b, `    <meta name="twitter:image" content="%s" />`, e(pic))
	out := titleTagRE.ReplaceAllLiteral(s.index, []byte(b.String()))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The page itself is never cached, so a new build is picked up at once.
	w.Header().Set("Cache-Control", "no-cache")
	if c.NotFound {
		// Still the app (it shows its own not-found page), but crawlers shouldn't keep it.
		w.Header().Set("X-Robots-Tag", "noindex")
	}
	w.Write(out)
}

// ServeImage writes the picture for the page in /og/<page>.png.
func (s *Sharer) ServeImage(w http.ResponseWriter, r *http.Request) {
	page, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/og"), ".png")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if page == "/home" {
		page = "/"
	}
	key := page
	if page == "/mods" {
		key += "?" + r.URL.RawQuery
	}
	s.mu.Lock()
	hit, found := s.cache[key]
	s.mu.Unlock()
	if !found || time.Now().After(hit.expires) {
		c := s.card(page, r.URL.Query())
		host := s.origin(r)
		host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
		hit = cachedCard{png: renderCard(c, host), expires: time.Now().Add(c.Fresh)}
		s.mu.Lock()
		if len(s.cache) >= ogCacheSize {
			clear(s.cache)
		}
		s.cache[key] = hit
		s.mu.Unlock()
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(time.Until(hit.expires).Seconds())))
	w.Write(hit.png)
}

var (
	matchPathRE  = regexp.MustCompile(`^/m/([A-Za-z0-9]{4,12})/?$`)
	fourPathRE   = regexp.MustCompile(`^/f/([A-Za-z0-9]{4,12})/?$`)
	replayPathRE = regexp.MustCompile(`^/g/([A-Za-z0-9]{4,12})/(\d{1,6})/?$`)
	userPathRE   = regexp.MustCompile(`^/u/([A-Za-z0-9_-]{1,20})/?$`)
)

const tagline = "Chess in the fog of war, where you win by capturing the king."

// card is the preview for a page of the app.
func (s *Sharer) card(path string, q url.Values) card {
	home := card{
		Title: "Eat the King",
		Lines: []string{"Chess in the fog of war.", "You see only what your pieces see. No check: win by capturing the king."},
		Board: startBoard(Settings{Fog: true}),
		Fresh: time.Hour,
	}
	if m := matchPathRE.FindStringSubmatch(path); m != nil {
		return s.gameCard(strings.ToUpper(m[1]), 0)
	}
	if m := replayPathRE.FindStringSubmatch(path); m != nil {
		round, _ := strconv.Atoi(m[2])
		return s.gameCard(strings.ToUpper(m[1]), max(round, 1))
	}
	if m := userPathRE.FindStringSubmatch(path); m != nil {
		return s.playerCard(m[1])
	}
	if fourPathRE.MatchString(path) {
		c := home
		c.Title = "A War Mode game"
		c.Lines = []string{"Four players in the fog on a cross-shaped board.", "Open the link to join or watch."}
		c.Fresh = 10 * time.Minute
		return c
	}
	switch strings.TrimSuffix(path, "/") {
	case "":
		return home
	case "/mods":
		return modsCard(parseModsLink(q))
	case "/rules":
		home.Title = "How to play"
		home.Lines = []string{"Chess in the fog of war: you see only the squares your pieces can reach.", "There is no check. You win by capturing the king."}
		return home
	case "/watch":
		home.Title = "Watch live games"
		home.Lines = []string{"Games being played right now, in the fog."}
		return home
	case "/login", "/signup", "/settings", "/friends", "/messages", "/tournaments", "/download":
		home.Lines = []string{tagline}
		return home
	}
	home.NotFound = true
	home.Lines = []string{tagline}
	return home
}

// gameCard is a match's preview: its latest round, or one round of it.
func (s *Sharer) gameCard(id string, round int) card {
	c := card{Title: "A game of Eat the King", Lines: []string{tagline}, Board: startBoard(Settings{Fog: true}), Fresh: 5 * time.Minute}
	rec, err := s.src.ShareGame(id, round)
	if err != nil {
		log.Printf("preview of game %s: %v", id, err)
		c.Fresh = time.Minute
		return c
	}
	if rec == nil {
		c.NotFound = true
		return c
	}
	names := [2]string{seatName(rec, white), seatName(rec, black)}
	mode := modeLine(rec.Settings)
	c.Board = startBoard(rec.Settings)
	c.Hill = rec.Settings.Hill
	switch {
	case rec.Result != "":
		c.Title = names[white] + " vs " + names[black]
		c.Lines = []string{resultLine(rec, names), mode, movesLine(rec)}
		// A finished game shows where it ended, everything in view.
		if r, err := buildReplay(rec); err == nil && len(r.Frames) > 0 {
			c.Board = dress(r.Frames[len(r.Frames)-1].Board, rec.Settings)
		}
		c.Fresh = 24 * time.Hour
	case rec.Seats[white] == "" || rec.Seats[black] == "":
		host := names[white]
		if rec.Seats[white] == "" {
			host = names[black]
		}
		c.Title = "Play me at Eat the King"
		if host != "Guest" {
			c.Title = host + " invites you to a game"
		}
		c.Lines = []string{mode, "Open the link to take the other seat.", tagline}
	default:
		c.Title = names[white] + " vs " + names[black]
		c.Lines = []string{"Being played now", mode}
	}
	if round > 0 && rec.Result != "" {
		c.Lines[2] = "Replay · " + c.Lines[2]
	}
	return c
}

// playerCard is a profile's preview.
func (s *Sharer) playerCard(name string) card {
	c := card{Title: name, Lines: []string{tagline}, Board: startBoard(Settings{Fog: true}), Fresh: time.Hour}
	p, err := s.src.SharePlayer(name)
	if err != nil {
		log.Printf("preview of player %s: %v", name, err)
		c.Fresh = time.Minute
		return c
	}
	if p == nil {
		c.NotFound = true
		return c
	}
	c.Title = p.Name
	var lines []string
	if t := titleNames[p.Title]; t != "" {
		lines = append(lines, t)
	}
	lines = append(lines, fmt.Sprintf("Rated %d in Eat the King", p.Rating))
	if p.FourRating > 0 {
		lines = append(lines, fmt.Sprintf("Rated %d in War Mode", p.FourRating))
	}
	if p.Games == 1 {
		lines = append(lines, "1 game played")
	} else {
		lines = append(lines, fmt.Sprintf("%d games played", p.Games))
	}
	c.Lines = lines
	return c
}

// titleNames are the titles in English. Must match titles.names in web/src/i18n/en.ts.
var titleNames = map[string]string{
	"club": "Club player", "expert": "Expert", "candidate": "Candidate master", "fog": "Fog master",
	"im": "International master", "gm": "Grandmaster", "king": "King",
}

func seatName(rec *gameRecord, c int) string {
	seat := rec.Seats[c]
	switch {
	case strings.HasPrefix(seat, "bot:"):
		if n := bots.Level(strings.TrimPrefix(seat, "bot:")).Name(); n != "" {
			return n
		}
		return "A bot"
	case c == white && rec.whiteName != "":
		return rec.whiteName
	case c == black && rec.blackName != "":
		return rec.blackName
	case seat == "":
		return "Open seat"
	}
	return "Guest"
}

func resultLine(rec *gameRecord, names [2]string) string {
	how := ""
	if rec.Reason != "" {
		how = " by " + rec.Reason
	}
	switch rec.Result {
	case engine.WhiteWins.String():
		return names[white] + " won" + how
	case engine.BlackWins.String():
		return names[black] + " won" + how
	}
	return "Drawn" + how
}

func movesLine(rec *gameRecord) string {
	n := (len(strings.Fields(rec.Moves)) + 1) / 2
	if rec.Settings.Sync {
		n = len(strings.Fields(rec.Moves))
	}
	if n == 1 {
		return "1 move"
	}
	return fmt.Sprintf("%d moves", n)
}

// modeLine is a game's kind and clock: "Eat the King · Blitz 5+0", "Mods: Prince, Ambush · No clock".
func modeLine(s Settings) string {
	kind := "Eat the King"
	if s.Bot != "" {
		kind = "Against " + bots.Level(s.Bot).Name()
	}
	if names := modNames(s); len(names) > 0 {
		kind = "Mods: " + strings.Join(names, ", ")
	} else if s.Mods {
		kind = "Mods"
	}
	return kind + " · " + clockName(s.Time)
}

// modNames are the mods a game was made with, in the order the Mods page lists them.
func modNames(s Settings) []string {
	var out []string
	add := func(on bool, name string) {
		if on {
			out = append(out, name)
		}
	}
	add(s.Sync, "Sync Move")
	add(s.JumpKing, "Prince")
	add(s.JesterPawns, "Jester pawns")
	add(s.Setup == "battleship", "Ambush")
	add(s.Setup == "backRow", "Back row")
	add(s.Hill, "King of the Hill")
	add(!s.Fog, "No fog")
	add(!s.KingCapture && !s.Sync, "Checkmate")
	add(s.Mods && s.CenterVision == "", "No centre vision")
	return out
}

// clockName mirrors describeTime in web/src/settings.ts.
func clockName(t TimeControl) string {
	const day = 24 * 60 * 60
	switch {
	case t.PerMove >= 2*day && t.PerMove%day == 0:
		return fmt.Sprintf("%d days per move", t.PerMove/day)
	case t.PerMove > 0:
		return fmt.Sprintf("%d h per move", (t.PerMove+1800)/3600)
	case !t.clocked():
		return "No clock"
	}
	group := "Classical"
	switch {
	case t == TimeControl{Base: 300} || t == TimeControl{Base: 180, Inc: 2}:
		group = "Blitz"
	case t == TimeControl{Base: 600}:
		group = "Rapid"
	case t == TimeControl{Base: 5400, Inc: 30}:
		group = "Tournament"
	case t.Base < 180:
		group = "Bullet"
	case t.Base < 600:
		group = "Blitz"
	case t.Base < 1800:
		group = "Rapid"
	}
	return fmt.Sprintf("%s %s+%d", group, strconv.FormatFloat(float64(t.Base)/60, 'f', -1, 64), t.Inc)
}

// startBoard is the usual start as White sees it: with fog, the far half hidden.
func startBoard(s Settings) string {
	b := []byte("RNBQKBNR" + "PPPPPPPP" + strings.Repeat(".", 32) + "pppppppp" + "rnbqkbnr")
	if s.Fog {
		for i := 32; i < 64; i++ {
			b[i] = '?'
		}
	}
	return dress(string(b), s)
}

// dress draws kings as princes and pawns as jesters in games with those mods.
func dress(board string, s Settings) string {
	b := []byte(board)
	for i, ch := range b {
		switch {
		case s.JumpKing && ch == 'K':
			b[i] = 'X'
		case s.JumpKing && ch == 'k':
			b[i] = 'x'
		case s.JesterPawns && ch == 'P':
			b[i] = 'J'
		case s.JesterPawns && ch == 'p':
			b[i] = 'j'
		}
	}
	return string(b)
}

// A mods link is /mods with the switches in its query, which the Mods page reads
// too (modsFromLink in web/src/share.ts):
//
//	/mods?mod=prince&mod=ambush&players=4&clock=10-5
//
// mod may repeat; clock is minutes-increment, 1d, 7d or none.
type modsLink struct {
	Settings Settings
	Four     bool
	Extended bool
}

// modKeys are the names of the mods in links, with the old names of two of them.
var modKeys = map[string]func(*modsLink){
	"prince":     func(l *modsLink) { l.Settings.JumpKing = true },
	"jump":       func(l *modsLink) { l.Settings.JumpKing = true },
	"ambush":     func(l *modsLink) { l.Settings.Setup = "battleship" },
	"battleship": func(l *modsLink) { l.Settings.Setup = "battleship" },
	"backrow":    func(l *modsLink) { l.Settings.Setup = "backRow" },
	"jester":     func(l *modsLink) { l.Settings.JesterPawns = true },
	"hill":       func(l *modsLink) { l.Settings.Hill = true },
	"sync":       func(l *modsLink) { l.Settings.Sync = true },
	"nofog":      func(l *modsLink) { l.Settings.Fog = false },
	"mate":       func(l *modsLink) { l.Settings.KingCapture = false },
	"nocenter":   func(l *modsLink) { l.Settings.CenterVision = "" },
	"extended":   func(l *modsLink) { l.Four, l.Extended = true, true },
}

func parseModsLink(q url.Values) modsLink {
	l := modsLink{Settings: Settings{Fog: true, KingCapture: true, CenterVision: "any", Time: TimeControl{Base: 300}, Mods: true}}
	for _, m := range q["mod"] {
		if set := modKeys[strings.ToLower(m)]; set != nil {
			set(&l)
		}
	}
	if l.Settings.Sync {
		l.Settings.KingCapture = true
	}
	if q.Get("players") == "4" {
		l.Four = true
	}
	if l.Extended {
		l.Settings.Hill = false
	}
	if t, ok := parseClock(q.Get("clock")); ok {
		l.Settings.Time = t
	}
	return l
}

func parseClock(s string) (TimeControl, bool) {
	switch s {
	case "none":
		return TimeControl{}, true
	case "1d":
		return TimeControl{PerMove: 24 * 60 * 60}, true
	case "7d":
		return TimeControl{PerMove: 7 * 24 * 60 * 60}, true
	}
	m, inc, ok := strings.Cut(s, "-")
	if !ok {
		return TimeControl{}, false
	}
	mins, err1 := strconv.Atoi(m)
	secs, err2 := strconv.Atoi(inc)
	if err1 != nil || err2 != nil || mins < 0 || secs < 0 || mins > 600 || secs > 600 {
		return TimeControl{}, false
	}
	return TimeControl{Base: mins * 60, Inc: secs}, true
}

// modsCard is a mods link's preview: the switches, and what each one does.
func modsCard(l modsLink) card {
	s := l.Settings
	names := modNames(s)
	if l.Extended {
		names = append([]string{"Extended board"}, names...)
	}
	c := card{Title: "Mods", Board: startBoard(s), Hill: s.Hill, Fresh: 24 * time.Hour}
	if len(names) > 0 {
		c.Title = "Mods: " + strings.Join(names, ", ")
	}
	players := "Two players"
	if l.Four {
		players = "Four players"
	}
	c.Lines = []string{players + " · " + clockName(s.Time)}
	add := func(on bool, line string) {
		if on {
			c.Lines = append(c.Lines, line)
		}
	}
	add(l.Extended, "Extended board: 16 by 16, four rows each, open corners.")
	add(s.Sync, "Sync Move: everyone moves at once.")
	add(s.JumpKing, "Prince: the king can also jump like a knight.")
	add(s.JesterPawns, "Jester pawns: pawns also step and take one square any way.")
	add(s.Setup == "battleship", "Ambush: place all your pieces anywhere in your rows.")
	add(s.Setup == "backRow", "Back row: arrange your back row first.")
	add(s.Hill, "King of the Hill: walk your king onto a centre square to win.")
	add(!s.Fog, "No fog: the whole board in plain sight.")
	add(!s.KingCapture && !s.Sync, "Checkmate: plain chess rules with check.")
	add(s.CenterVision == "", "No centre vision.")
	if len(names) == 0 {
		c.Lines = append(c.Lines, "Every rule is a switch. Open the link to pick yours.")
	}
	return c
}

// spa serves the built web app: files as they are, and index.html, with the
// page's preview tags, for every other path so /m/ABC123 deep links work.
func spa(sub fs.FS, share *Sharer) http.Handler {
	files := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			share.ServePage(w, r)
			return
		}
		if _, err := fs.Stat(sub, path); err != nil {
			share.ServePage(w, r)
			return
		}
		switch {
		case strings.HasPrefix(path, "assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case path == "sw.js" || path == "manifest.webmanifest":
			// A new service worker or manifest must reach phones at once.
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

// indexHTML is the built app's index.html, or a stand-in when there is none.
func indexHTML(sub fs.FS) []byte {
	b, err := fs.ReadFile(sub, "index.html")
	if err != nil || !bytes.Contains(b, []byte("<title>")) {
		return []byte("<!doctype html><html><head><title>Eat the King</title></head><body></body></html>")
	}
	return b
}
