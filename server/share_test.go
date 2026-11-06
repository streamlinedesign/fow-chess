package main

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

type fakeShare struct {
	games   map[string]*gameRecord
	players map[string]*sharePlayer
}

func (f fakeShare) ShareGame(id string, round int) (*gameRecord, error) {
	return f.games[id], nil
}

func (f fakeShare) SharePlayer(name string) (*sharePlayer, error) {
	return f.players[strings.ToLower(name)], nil
}

func testSharer() *Sharer {
	plain := Settings{Fog: true, KingCapture: true, CenterVision: "any", Time: TimeControl{Base: 300}}
	src := fakeShare{
		games: map[string]*gameRecord{
			"LIVE12": {MatchID: "LIVE12", Round: 1, Settings: plain, Seats: [2]string{"u:1", "u:2"}, Moves: "e2e4 e7e5 g1f3",
				whiteName: "alice", blackName: "bob"},
			"OPEN12": {MatchID: "OPEN12", Round: 1, Settings: plain, Seats: [2]string{"u:1", ""}, whiteName: "alice"},
			"DONE12": {MatchID: "DONE12", Round: 1, Settings: Settings{Fog: true, KingCapture: true, JumpKing: true, Mods: true, CenterVision: "any", Time: TimeControl{Base: 600, Inc: 5}},
				Seats: [2]string{"u:1", "bot:solomon"}, Moves: "e2e4 e7e5", Result: "1-0", Reason: "resignation", whiteName: "alice"},
		},
		players: map[string]*sharePlayer{"alice": {Name: "alice", Rating: 1320, Title: "expert", Games: 42}},
	}
	index := []byte(`<!doctype html><html><head><title>Eat the King</title></head><body></body></html>`)
	return NewSharer(src, "https://example.com/", index)
}

func page(t *testing.T, s *Sharer, path string) string {
	t.Helper()
	w := httptest.NewRecorder()
	s.ServePage(w, httptest.NewRequest("GET", path, nil))
	return w.Body.String()
}

func TestPreviewTags(t *testing.T) {
	s := testSharer()
	cases := []struct {
		path string
		want []string
	}{
		{"/", []string{`<title>Eat the King</title>`, `og:image" content="https://example.com/og/home.png"`, `twitter:card" content="summary_large_image"`}},
		{"/m/live12", []string{`alice vs bob · Eat the King`, `Being played now`, `og:image" content="https://example.com/og/m/live12.png"`}},
		{"/m/OPEN12", []string{`alice invites you to a game`, `Eat the King · Blitz 5+0`}},
		{"/m/DONE12", []string{`alice vs Solomon`, `alice won by resignation`, `Mods: Prince · Rapid 10+5`}},
		{"/g/DONE12/1", []string{`Replay · 1 move`}},
		{"/u/alice", []string{`<title>alice · Eat the King`, `Expert · Rated 1320 in Eat the King · 42 games played`}},
		{"/mods?mod=prince&mod=opensetup&players=4&clock=10-5", []string{
			`Mods: Prince, Open Setup`, `Four players · Rapid 10+5`,
			`og:image" content="https://example.com/og/mods.png?mod=prince&amp;mod=opensetup&amp;players=4&amp;clock=10-5"`,
		}},
		// Links made while it was called Ambush still work.
		{"/mods?mod=ambush", []string{`Mods: Open Setup`}},
		{"/nope", []string{`<title>Eat the King</title>`}},
		{"/m/<script>", []string{`<title>Eat the King</title>`}},
	}
	for _, c := range cases {
		got := page(t, s, c.path)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: missing %q in\n%s", c.path, w, got)
			}
		}
		if strings.Count(got, "<title>") != 1 {
			t.Errorf("%s: want one title, got\n%s", c.path, got)
		}
	}
}

// A preview must never show a game that's still being played: that would lift the fog.
func TestPreviewKeepsTheFog(t *testing.T) {
	s := testSharer()
	live := s.card("/m/LIVE12", nil)
	if live.Board != startBoard(Settings{Fog: true}) {
		t.Errorf("live game shows %q", live.Board)
	}
	done := s.card("/m/DONE12", nil)
	if strings.Contains(done.Board, "?") || done.Board[28] != 'P' || !strings.Contains(done.Board, "X") {
		t.Errorf("finished game should show the final position with princes, got %q", done.Board)
	}
}

func TestModsLink(t *testing.T) {
	q, _ := url.ParseQuery("mod=jump&mod=battleship&mod=sync&mod=mate&mod=hill&mod=extended&clock=none&mod=bogus")
	l := parseModsLink(q)
	s := l.Settings
	if !s.JumpKing || s.Setup != "battleship" || !s.Sync || !s.KingCapture || s.Hill || !l.Four || !l.Extended || s.Time != (TimeControl{}) {
		t.Errorf("got %+v", l)
	}
	for in, want := range map[string]TimeControl{"1d": {PerMove: 86400}, "3-2": {Base: 180, Inc: 2}, "x": {Base: 300}, "9999-0": {Base: 300}} {
		q := url.Values{"clock": {in}}
		if got := parseModsLink(q).Settings.Time; got != want {
			t.Errorf("clock %q: got %+v, want %+v", in, got, want)
		}
	}
}

func TestClockName(t *testing.T) {
	for tc, want := range map[TimeControl]string{
		{}: "No clock", {Base: 300}: "Blitz 5+0", {Base: 60}: "Bullet 1+0", {Base: 5400, Inc: 30}: "Tournament 90+30",
		{Base: 1800, Inc: 20}: "Classical 30+20", {PerMove: 86400}: "24 h per move", {PerMove: 7 * 86400}: "7 days per move",
	} {
		if got := clockName(tc); got != want {
			t.Errorf("%+v: got %q, want %q", tc, got, want)
		}
	}
}

func TestPreviewImage(t *testing.T) {
	s := testSharer()
	for _, path := range []string{"/og/home.png", "/og/m/DONE12.png", "/og/u/alice.png", "/og/mods.png?mod=hill&mod=jester&mod=nofog"} {
		w := httptest.NewRecorder()
		s.ServeImage(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s: %d %s", path, w.Code, w.Header().Get("Content-Type"))
		}
		img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
		if err != nil || img.Bounds().Dx() != ogW || img.Bounds().Dy() != ogH {
			t.Fatalf("%s: %v", path, err)
		}
		// OG_OUT=dir writes the pictures out to look at.
		if dir := os.Getenv("OG_OUT"); dir != "" {
			name := strings.NewReplacer("/", "_", "?", "_", "&", "_", "=", "-").Replace(strings.TrimPrefix(path, "/og/"))
			os.WriteFile(dir+"/"+strings.TrimSuffix(name, ".png")+".png", w.Body.Bytes(), 0o644)
		}
	}
	// The second time comes from the cache.
	start := time.Now()
	w := httptest.NewRecorder()
	s.ServeImage(w, httptest.NewRequest("GET", "/og/home.png", nil))
	if time.Since(start) > 50*time.Millisecond {
		t.Errorf("cached picture took %v", time.Since(start))
	}
}

func TestSPAServesFilesAndPages(t *testing.T) {
	sub := fstest.MapFS{
		"index.html":           {Data: []byte(`<html><head><title>Eat the King</title></head></html>`)},
		"sw.js":                {Data: []byte(`self`)},
		"assets/app-1.js":      {Data: []byte(`x`)},
		"manifest.webmanifest": {Data: []byte(`{}`)},
	}
	h := spa(sub, NewSharer(fakeShare{}, "", indexHTML(sub)))
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	if w := get("/sw.js"); w.Body.String() != "self" || w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("sw.js: %q %q", w.Body.String(), w.Header().Get("Cache-Control"))
	}
	if w := get("/assets/app-1.js"); !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("assets not cached for long")
	}
	if w := get("/rules"); !strings.Contains(w.Body.String(), "How to play") || !strings.Contains(w.Body.String(), `og:url" content="http://example.com/rules"`) {
		t.Errorf("rules page: %s", w.Body.String())
	}
}
