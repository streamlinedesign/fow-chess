package main

import (
	"chess/bots"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"math/rand/v2"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websocket"
)

// dist holds the built web app; `make build` copies web/dist here.
//
//go:embed all:dist
var dist embed.FS

var verbose = flag.Bool("v", false, "log every match action with how long it took")

func main() {
	addr := flag.String("addr", ":"+envOr("PORT", "4100"), "listen address (or set PORT); production uses -addr 127.0.0.1:4100 behind Caddy")
	dbURL := flag.String("db", envOr("DATABASE_URL", defaultDatabaseURL), "Postgres connection URL for accounts and games (or set DATABASE_URL)")
	migrateOnly := flag.Bool("migrate", false, "create the database if needed, apply migrations, and exit")
	bugs := flag.Int("bugs", 0, "print the newest N bug reports and exit")
	flag.Parse()
	if *verbose {
		log.Printf("verbose: logging every match action with timings")
	}

	store, err := OpenStore(*dbURL)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	if *migrateOnly {
		store.Close()
		log.Printf("database %s is up to date", redactURL(*dbURL))
		return
	}
	if *bugs > 0 {
		err := printBugReports(os.Stdout, store, *bugs)
		store.Close()
		if err != nil {
			log.Fatalf("reading bug reports: %v", err)
		}
		return
	}
	hub := NewHub(store)
	auth := &Auth{store: store, limiter: newFailLimiter(), signups: newLimiter(maxSignups, time.Hour)}
	creates := newLimiter(maxCreates, limitWindow)
	misses := newLimiter(maxMisses, limitWindow)
	live := NewLive(store)
	social := &Social{store: store, live: live, hub: hub, auth: auth, queue: NewQueue()}
	live.playing = hub.Playing
	tournaments := NewTournaments(store, hub, live, social.notify)
	hub.SetHooks(matchHooks{notify: social.notify, playing: live.presence, over: tournaments.gameOver})
	go tournaments.run()
	mux := http.NewServeMux()
	auth.routes(mux)
	profileRoutes(mux, store, auth, live)
	social.routes(mux)
	social.watchRoutes(mux)
	social.queueRoutes(mux)
	social.tournamentRoutes(mux, tournaments)
	bugRoutes(mux, store, auth)
	unlockRoutes(mux, store, auth)

	mux.HandleFunc("POST /api/matches", func(w http.ResponseWriter, r *http.Request) {
		// The creator gets white even if the opponent's browser connects first.
		if !creates.take(clientIP(r)) {
			jsonError(w, "Too many attempts. Try again in a few minutes.", http.StatusTooManyRequests)
			return
		}
		p := playerFor(r, auth.user(r))
		if p == nil {
			p = &player{}
		}
		// Settings come as an optional JSON body; an empty body means the
		// standard start with no clock.
		var req struct {
			Settings
			// Side is the colour the player wants against a bot: "w", "b", or "" for
			// either. The demo board defaults to White.
			Side string `json:"side"`
		}
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req)
		settings := req.Settings
		settings.Tournament = 0 // only the arena seats arena games
		settings.normalize()
		if !settings.valid() {
			jsonError(w, "Those match settings aren't available.", http.StatusBadRequest)
			return
		}
		var m *Match
		if settings.Bot != "" {
			// Against a bot you get the colour you prefer, or a random one so you
			// learn both. The demo board starts you as White.
			bot := botPlayer(bots.Level(settings.Bot))
			if req.Side == "b" {
				m = hub.CreateBetween(bot, p, settings)
			} else if req.Side == "w" || settings.Demo || rand.IntN(2) == 0 {
				m = hub.CreateBetween(p, bot, settings)
			} else {
				m = hub.CreateBetween(bot, p, settings)
			}
		} else {
			m = hub.Create(p, settings)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": m.id})
	})

	mux.HandleFunc("GET /api/matches/{id}", func(w http.ResponseWriter, r *http.Request) {
		if findMatch(w, r, misses, hub.Get) != nil {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	mux.HandleFunc("GET /ws/{id}", func(w http.ResponseWriter, r *http.Request) {
		p := playerFor(r, auth.userForSocket(r))
		if p == nil {
			http.NotFound(w, r)
			return
		}
		m := findMatch(w, r, misses, hub.Get)
		if m == nil {
			return
		}
		// Skip the library's same-origin check: dev proxies and LAN IPs (a phone
		// on 192.168.x.x) make Origin and Host disagree. The session cookie only
		// counts for our own origin (userForSocket), so a cross-origin page plays
		// as a guest and can't act as a signed-in user.
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		m.serve(r.Context(), conn, p)
	})

	// Four-player games, from the mods page.
	fourHub := NewFourHub(store)
	social.fours, social.fourQueue = fourHub, NewFourQueue()
	social.fourQueueRoutes(mux)
	mux.HandleFunc("POST /api/four", func(w http.ResponseWriter, r *http.Request) {
		if !creates.take(clientIP(r)) {
			jsonError(w, "Too many attempts. Try again in a few minutes.", http.StatusTooManyRequests)
			return
		}
		p := playerFor(r, auth.user(r))
		if p == nil {
			p = &player{}
		}
		var settings fourSettings
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&settings)
		settings.Rated = false // only the queue starts ranked games
		if settings.Demo {
			settings = demoFour(bots.Level(settings.Bot))
		} else if settings.War {
			// War Mode with a friend: its rules, and one of its clocks.
			if !slices.Contains(warTimes, settings.Time) {
				jsonError(w, "Those match settings aren't available.", http.StatusBadRequest)
				return
			}
			settings = warFour(settings.Time)
		}
		if !settings.valid() {
			jsonError(w, "Those match settings aren't available.", http.StatusBadRequest)
			return
		}
		m := fourHub.Create(p, settings)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": m.id})
	})
	mux.HandleFunc("GET /api/four/{id}", func(w http.ResponseWriter, r *http.Request) {
		if findMatch(w, r, misses, fourHub.Get) != nil {
			w.WriteHeader(http.StatusNoContent)
		}
	})
	mux.HandleFunc("GET /ws/four/{id}", func(w http.ResponseWriter, r *http.Request) {
		p := playerFor(r, auth.userForSocket(r))
		if p == nil {
			http.NotFound(w, r)
			return
		}
		m := findMatch(w, r, misses, fourHub.Get)
		if m == nil {
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		m.serve(r.Context(), conn, p)
	})

	mux.Handle("/", spa(dist))

	srv := &http.Server{Addr: *addr, Handler: mux}
	go func() {
		// On Ctrl-C or a service stop, finish writing queued games before exiting.
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("chess listening on %s (database %s)", *addr, redactURL(*dbURL))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	store.Close()
}

// Rate limits per address (see clientIP), so the beta can't be flooded with
// accounts or matches, and nobody finds open games by trying codes: there are
// 31^6 codes, and an address gets maxMisses wrong ones per limitWindow.
const (
	maxSignups  = 10 // per hour
	maxCreates  = 60
	maxMisses   = 60
	limitWindow = 10 * time.Minute
)

// findMatch looks up the match whose code is in the URL, answering 404 for
// none and 429 for an address that has tried too many codes that don't exist.
func findMatch[M any](w http.ResponseWriter, r *http.Request, misses *failLimiter, get func(string) *M) *M {
	key := clientIP(r)
	if !misses.allow(key) {
		jsonError(w, "Too many attempts. Try again in a few minutes.", http.StatusTooManyRequests)
		return nil
	}
	m := get(strings.ToUpper(r.PathValue("id")))
	if m == nil {
		misses.fail(key)
		http.NotFound(w, r)
	}
	return m
}

// guestIDRE is what a browser's guest ID may look like. Seat keys of signed-in
// players ("u:<id>") and bots ("bot:<level>") have a colon, so no guest ID can
// name one of their seats and play it.
var guestIDRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// playerFor is who a request comes from: the signed-in user, else the guest
// ID the browser sends. Nil if there's neither.
func playerFor(r *http.Request, u *User) *player {
	browserID := r.URL.Query().Get("player")
	if !guestIDRE.MatchString(browserID) {
		browserID = ""
	}
	if u != nil {
		return userPlayer(u, browserID)
	}
	if browserID == "" {
		return nil
	}
	return guestPlayer(browserID)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// redactURL hides the password in a connection URL so it can be logged.
func redactURL(s string) string {
	if u, err := url.Parse(s); err == nil {
		return u.Redacted()
	}
	return "(unparseable URL)"
}

// spa serves static files and falls back to index.html so /m/ABC123 deep links work.
func spa(fsys embed.FS) http.Handler {
	// The web app manifest, so phones offer to install the site.
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
	sub, _ := fs.Sub(fsys, "dist")
	files := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if _, err := fs.Stat(sub, path); path != "" && err != nil {
			r.URL.Path = "/"
		}
		if strings.HasPrefix(path, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}
