package main

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMatchSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chess.db")
	store := openTestStore(t, path)
	hub := NewHub(store)
	alice, bob := guestPlayer("alice-browser"), guestPlayer("bob-browser")
	m := hub.Create(alice, Settings{Fog: true})

	cw, cb := newClient(alice.key), newClient(bob.key)
	m.join(cw, alice)
	m.join(cb, bob)
	for i, uci := range []string{"e2e4", "e7e5", "g1f3"} {
		c := cw
		if i%2 == 1 {
			c = cb
		}
		m.handle(c, inMsg{Type: "move", UCI: uci})
	}
	m.handle(cb, inMsg{Type: "resign"})
	store.Close() // flushes queued writes

	store = openTestStore(t, path)
	defer store.Close()
	got := NewHub(store).Get(m.id)
	if got == nil {
		t.Fatal("match not restored")
	}
	if got.seats != [2]string{"alice-browser", "bob-browser"} || !got.settings.Fog {
		t.Errorf("seats %v settings %+v", got.seats, got.settings)
	}
	if n := len(got.game.History()); n != 3 {
		t.Errorf("restored %d moves, want 3", n)
	}
	if got.game.Outcome().String() != "1-0" || got.game.Method().String() != "resignation" {
		t.Errorf("result %s by %s", got.game.Outcome(), got.game.Method())
	}
}

func TestGuestKeepsSeatAfterSigningIn(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "chess.db"))
	defer store.Close()
	hub := NewHub(store)
	guest := guestPlayer("browser-1")
	m := hub.Create(guest, Settings{})
	u, err := store.CreateUser("Carol", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	m.join(newClient("u:1"), userPlayer(u, "browser-1"))
	if m.seats[white] != "u:1" || m.names[white] != "Carol" || m.seats[black] != "" {
		t.Errorf("seats %v names %v", m.seats, m.names)
	}
}

func TestAccounts(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "chess.db"))
	defer store.Close()
	mux := http.NewServeMux()
	auth := &Auth{store: store, limiter: newFailLimiter()}
	auth.routes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	post := func(path, body string) int {
		t.Helper()
		res, err := client.Post(srv.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	// me reports whether the client is signed in.
	me := func() bool {
		res, err := client.Get(srv.URL + "/api/me")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var u *meResponse
		json.NewDecoder(res.Body).Decode(&u)
		return u != nil
	}

	if code := post("/api/signup", `{"username":"Dana","password":"short"}`); code != 400 {
		t.Errorf("short password: %d", code)
	}
	if code := post("/api/signup", `{"username":"no spaces","password":"long enough"}`); code != 400 {
		t.Errorf("bad username: %d", code)
	}
	if code := post("/api/signup", `{"username":"Dana","password":"long enough"}`); code != 200 {
		t.Fatalf("signup: %d", code)
	}
	if !me() {
		t.Error("not signed in after signup")
	}
	if code := post("/api/signup", `{"username":"dana","password":"long enough"}`); code != 409 {
		t.Errorf("same name, other case: %d", code)
	}
	if code := post("/api/logout", ``); code != 204 {
		t.Errorf("logout: %d", code)
	}
	if me() {
		t.Error("still signed in after logout")
	}
	if code := post("/api/login", `{"username":"DANA","password":"wrong password"}`); code != 401 {
		t.Errorf("wrong password: %d", code)
	}
	if code := post("/api/login", `{"username":"DANA","password":"long enough"}`); code != 200 {
		t.Errorf("login: %d", code)
	}
	if !me() {
		t.Error("not signed in after login")
	}

	// A form post from another site can't send JSON.
	res, _ := client.Post(srv.URL+"/api/login", "text/plain", strings.NewReader(`{"username":"dana","password":"long enough"}`))
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain login: %d", res.StatusCode)
	}
}

func TestLoginLimiter(t *testing.T) {
	l := newFailLimiter()
	for range maxFails {
		if !l.allow("1.2.3.4") {
			t.Fatal("blocked too early")
		}
		l.fail("1.2.3.4")
	}
	if l.allow("1.2.3.4") || !l.allow("5.6.7.8") {
		t.Error("limiter should block only the failing address")
	}
}
