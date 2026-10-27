package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

// dist holds the built web app; `make build` copies web/dist here.
//
//go:embed all:dist
var dist embed.FS

var verbose = flag.Bool("v", false, "log every match action with how long it took")

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()
	if *verbose {
		log.Printf("verbose: logging every match action with timings")
	}

	hub := NewHub()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/matches", func(w http.ResponseWriter, r *http.Request) {
		// The creator gets white even if the opponent's browser connects first.
		player := r.URL.Query().Get("player")
		if len(player) > 64 {
			player = ""
		}
		// Settings come as an optional JSON body; an empty body means standard chess.
		var settings Settings
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&settings)
		m := hub.Create(player, settings)
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
		player := r.URL.Query().Get("player")
		if m == nil || player == "" || len(player) > 64 {
			http.NotFound(w, r)
			return
		}
		// Skip the same-origin check: dev proxies and LAN IPs (a phone on
		// 192.168.x.x) make Origin and Host disagree, and the socket carries no
		// cookies, so a cross-origin page can't do more than any other client.
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		m.serve(r.Context(), conn, player)
	})

	mux.Handle("/", spa(dist))

	log.Printf("chess listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
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
