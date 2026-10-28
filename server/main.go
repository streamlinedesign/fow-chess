package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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
	addr := flag.String("addr", ":8080", "listen address")
	dbURL := flag.String("db", envOr("DATABASE_URL", defaultDatabaseURL), "Postgres connection URL for accounts and games (or set DATABASE_URL)")
	migrateOnly := flag.Bool("migrate", false, "create the database if needed, apply migrations, and exit")
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
	hub := NewHub(store)
	auth := &Auth{store: store, limiter: newFailLimiter()}
	live := NewLive(store)
	social := &Social{store: store, live: live, hub: hub, auth: auth}
	live.playing = hub.Playing
	hub.SetHooks(matchHooks{notify: social.notify, playing: live.presence})
	mux := http.NewServeMux()
	auth.routes(mux)
	profileRoutes(mux, store, auth, live)
	social.routes(mux)
	social.watchRoutes(mux)

	// playerFor is who a request comes from: the signed-in user, else the guest
	// ID the browser sends. Nil if there's neither.
	playerFor := func(r *http.Request, u *User) *player {
		browserID := r.URL.Query().Get("player")
		if len(browserID) > 64 {
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

	mux.HandleFunc("POST /api/matches", func(w http.ResponseWriter, r *http.Request) {
		// The creator gets white even if the opponent's browser connects first.
		p := playerFor(r, auth.user(r))
		if p == nil {
			p = &player{}
		}
		// Settings come as an optional JSON body; an empty body means the
		// standard start with no clock.
		var settings Settings
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&settings)
		settings.normalize()
		if !settings.valid() {
			jsonError(w, "Those match settings aren't available.", http.StatusBadRequest)
			return
		}
		m := hub.Create(p, settings)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": m.id})
	})

	mux.HandleFunc("GET /api/matches/{id}", func(w http.ResponseWriter, r *http.Request) {
		if hub.Get(strings.ToUpper(r.PathValue("id"))) == nil {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /ws/{id}", func(w http.ResponseWriter, r *http.Request) {
		m := hub.Get(strings.ToUpper(r.PathValue("id")))
		p := playerFor(r, auth.userForSocket(r))
		if m == nil || p == nil {
			http.NotFound(w, r)
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
