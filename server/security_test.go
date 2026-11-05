package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A guest ID is whatever the browser sends, so it must never be able to name
// a signed-in player's seat ("u:<id>") or a bot's ("bot:<level>").
func TestGuestCantTakeAnotherSeat(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "chess.db"))
	hub := NewHub(store)
	alice, _ := store.CreateUser("alice", dummyHash)
	bob, _ := store.CreateUser("bob", dummyHash)
	mallory, _ := store.CreateUser("mallory", dummyHash)
	ids := []string{"u:" + strconv.FormatInt(alice.ID, 10), "u:" + strconv.FormatInt(bob.ID, 10), "bot:solomon", "a b", "x\x00"}
	m := hub.CreateBetween(userPlayer(alice, "a-browser"), userPlayer(bob, "b-browser"), Settings{})

	for _, id := range ids {
		r := httptest.NewRequest("GET", "/ws/"+m.id+"?player="+url.QueryEscape(id), nil)
		if p := playerFor(r, nil); p != nil {
			t.Errorf("guest ID %q was accepted as %q", id, p.key)
		}
		// A signed-in player can't claim a seat as "their" guest seat either.
		p := playerFor(r, mallory)
		c := newClient(p.key)
		m.join(c, p)
		m.handle(c, inMsg{Type: "resign"})
		m.leave(c)
		if m.colorOf(p.key) >= 0 {
			t.Fatalf("mallory took a seat with guest ID %q", id)
		}
	}
	if m.game.Over() {
		t.Error("someone resigned for a player")
	}

	r := httptest.NewRequest("GET", "/ws/x?player=0123456789abcdef0123456789abcdef", nil)
	if p := playerFor(r, nil); p == nil || p.key != "0123456789abcdef0123456789abcdef" {
		t.Errorf("a browser's own ID was refused: %+v", p)
	}
}

func TestClientIP(t *testing.T) {
	for _, c := range []struct {
		remote, xff, want string
	}{
		{"203.0.113.7:5000", "", "203.0.113.7"},
		// Only a proxy on this machine is believed.
		{"203.0.113.7:5000", "198.51.100.1", "203.0.113.7"},
		{"127.0.0.1:5000", "198.51.100.1", "198.51.100.1"},
		{"127.0.0.1:5000", "10.0.0.1, 198.51.100.1", "198.51.100.1"},
		{"[::1]:5000", "198.51.100.1", "198.51.100.1"},
		{"127.0.0.1:5000", "", "127.0.0.1"},
		{"127.0.0.1:5000", "not an address", "127.0.0.1"},
		// An IPv6 client counts as its /64, which one machine usually has to itself.
		{"127.0.0.1:5000", "2001:db8:1:2:aaaa::1", "2001:db8:1:2::/64"},
		{"[2001:db8:1:2:bbbb::9]:5000", "", "2001:db8:1:2::/64"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := clientIP(r); got != c.want {
			t.Errorf("%s via %q: got %s, want %s", c.remote, c.xff, got, c.want)
		}
	}
}

// Behind Caddy, failed logins from one address mustn't lock out everyone else.
func TestLoginLimitIsPerClient(t *testing.T) {
	site := newTestSite(t)
	site.signup("carol")
	try := func(ip, password string) int {
		req, _ := http.NewRequest("POST", site.srv.URL+"/api/login", strings.NewReader(`{"username":"carol","password":"`+password+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", ip)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	for range maxFails {
		try("198.51.100.1", "wrong guess")
	}
	if code := try("198.51.100.1", "long enough"); code != http.StatusTooManyRequests {
		t.Errorf("attacker after %d fails: %d, want 429", maxFails, code)
	}
	if code := try("198.51.100.2", "long enough"); code != http.StatusOK {
		t.Errorf("someone else signing in: %d, want 200", code)
	}
}

func TestLimiter(t *testing.T) {
	l := newLimiter(3, time.Minute)
	for range 3 {
		if !l.allow("a") {
			t.Fatal("refused too soon")
		}
		l.fail("a")
	}
	if l.allow("a") {
		t.Error("allowed a fourth")
	}
	if !l.allow("b") {
		t.Error("another key was refused")
	}
	if !l.take("b") || !l.take("b") || !l.take("b") || l.take("b") {
		t.Error("take should allow three, then refuse")
	}
}

// Trying codes runs out quickly; a code that exists never counts against you.
func TestFindMatchLimitsMisses(t *testing.T) {
	misses := newLimiter(3, time.Minute)
	real := &Match{id: "ABC234"}
	get := func(id string) *Match {
		if id == real.id {
			return real
		}
		return nil
	}
	find := func(code, ip string) int {
		r := httptest.NewRequest("GET", "/api/matches/"+code, nil)
		r.SetPathValue("id", code)
		r.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		findMatch(w, r, misses, get)
		return w.Code
	}
	for range 5 {
		if code := find("abc234", "203.0.113.7"); code != http.StatusOK {
			t.Fatalf("real code: %d", code)
		}
	}
	for range 3 {
		if code := find("ZZZZZZ", "203.0.113.7"); code != http.StatusNotFound {
			t.Fatalf("wrong code: %d, want 404", code)
		}
	}
	if code := find("ABC234", "203.0.113.7"); code != http.StatusTooManyRequests {
		t.Errorf("after 3 misses: %d, want 429", code)
	}
	if code := find("ABC234", "203.0.113.8"); code != http.StatusOK {
		t.Errorf("another address: %d, want 200", code)
	}
}
