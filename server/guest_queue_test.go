package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// Guests queue for one unranked game each: two for Eat the King at Rapid
// 10+0, four for War Mode. Signed-in players can't join.
func TestGuestQueue(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	auth := &Auth{store: store, limiter: newFailLimiter()}
	social := &Social{store: store, hub: NewHub(store), auth: auth, fours: NewFourHub(store), guestQueue: NewGuestQueue()}
	mux := http.NewServeMux()
	auth.routes(mux)
	social.guestQueueRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	call := func(method, guest, body string) (int, map[string]any) {
		t.Helper()
		req, _ := http.NewRequest(method, srv.URL+"/api/guest/queue?player="+guest, strings.NewReader(body))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	if code, _ := call("POST", "g1", `{"game":"blitz"}`); code != 400 {
		t.Errorf("an unknown game: %d", code)
	}
	if _, b := call("POST", "g1", `{"game":"etk"}`); b["waiting"] != true {
		t.Fatalf("first guest: %v", b)
	}
	// A War Mode guest doesn't make up an Eat the King pair.
	call("POST", "w1", `{"game":"war"}`)
	if _, b := call("GET", "g1", ""); b["waiting"] != true {
		t.Fatalf("still alone: %v", b)
	}
	_, b := call("POST", "g2", `{"game":"etk"}`)
	id, _ := b["matchId"].(string)
	if id == "" {
		t.Fatalf("second guest: %v", b)
	}
	if _, b := call("GET", "g1", ""); b["matchId"] != id {
		t.Errorf("the first guest is told %v, want %s", b, id)
	}
	m := social.hub.Get(id)
	if m == nil {
		t.Fatal("no match")
	}
	m.mu.Lock()
	if m.settings.Time != guestRapid || !m.settings.Unrated || !m.settings.Fog || !m.settings.KingCapture {
		t.Errorf("guest game settings: %+v", m.settings)
	}
	m.mu.Unlock()

	for _, g := range []string{"w2", "w3"} {
		if _, b := call("POST", g, `{"game":"war"}`); b["waiting"] != true {
			t.Fatalf("%s: %v", g, b)
		}
	}
	_, b = call("POST", "w4", `{"game":"war"}`)
	fid, _ := b["matchId"].(string)
	f := social.fours.Get(fid)
	if f == nil {
		t.Fatalf("four guests: %v", b)
	}
	f.mu.Lock()
	if f.settings.Rated || !f.settings.War || f.settings.Time != warTimes[0] {
		t.Errorf("guest War Mode settings: %+v", f.settings)
	}
	f.mu.Unlock()

	if code, _ := call("DELETE", "w1", ""); code != 204 {
		t.Errorf("leaving: %d", code)
	}
}
