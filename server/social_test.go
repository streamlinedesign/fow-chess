package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testSite is the whole API on a test server, backed by its own schema.
type testSite struct {
	t     *testing.T
	srv   *httptest.Server
	store *Store
	hub   *Hub
}

func newTestSite(t *testing.T) *testSite {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	hub := NewHub(store)
	auth := &Auth{store: store, limiter: newFailLimiter()}
	live := NewLive(store)
	social := &Social{store: store, live: live, hub: hub, auth: auth}
	hub.SetNotify(social.notify)
	mux := http.NewServeMux()
	auth.routes(mux)
	profileRoutes(mux, store, auth, live)
	social.routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &testSite{t, srv, store, hub}
}

// testUser is a signed-in browser.
type testUser struct {
	site   *testSite
	client *http.Client
	name   string
}

func (s *testSite) signup(name string) *testUser {
	jar, _ := cookiejar.New(nil)
	u := &testUser{s, &http.Client{Jar: jar}, name}
	if code, _ := u.do("POST", "/api/signup", `{"username":"`+name+`","password":"long enough"}`); code != 200 {
		s.t.Fatalf("signup %s: %d", name, code)
	}
	return u
}

// do sends a request and returns the status and the decoded JSON body.
func (u *testUser) do(method, path, body string) (int, map[string]any) {
	u.site.t.Helper()
	req, _ := http.NewRequest(method, u.site.srv.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := u.client.Do(req)
	if err != nil {
		u.site.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var out map[string]any
	if json.Unmarshal(b, &out) != nil {
		// Lists come back wrapped so callers can index them the same way.
		var list []any
		if json.Unmarshal(b, &list) == nil {
			out = map[string]any{"list": list}
		}
	}
	return res.StatusCode, out
}

func (u *testUser) notifications() []map[string]any {
	u.site.t.Helper()
	// Notifications are written off the request, so give them a moment.
	time.Sleep(50 * time.Millisecond)
	_, body := u.do("GET", "/api/notifications", "")
	var out []map[string]any
	for _, n := range body["items"].([]any) {
		out = append(out, n.(map[string]any))
	}
	return out
}

func kinds(ns []map[string]any) []string {
	var out []string
	for _, n := range ns {
		out = append(out, n["kind"].(string))
	}
	return out
}

func TestFriendsAndMessages(t *testing.T) {
	site := newTestSite(t)
	ann, ben, cat := site.signup("ann"), site.signup("ben"), site.signup("cat")

	_, res := ann.do("GET", "/api/users/search?q=BE", "")
	if list := res["list"].([]any); len(list) != 1 || list[0].(map[string]any)["username"] != "ben" || list[0].(map[string]any)["friend"] != "none" {
		t.Errorf("search: %v", res)
	}

	if _, res := ann.do("POST", "/api/friends/ben", ""); res["friend"] != "sent" {
		t.Errorf("request: %v", res)
	}
	ns := ben.notifications()
	if len(ns) != 1 || ns[0]["kind"] != notifyFriendRequest || ns[0]["actor"] != "ann" || ns[0]["pending"] != true {
		t.Errorf("ben's bell: %v", ns)
	}
	if code, _ := ben.do("POST", "/api/messages/ann", `{"text":"hi"}`); code != 403 {
		t.Errorf("message before being friends: %d", code)
	}
	if _, res := ben.do("POST", "/api/friends/ann", ""); res["friend"] != "friends" {
		t.Errorf("accept: %v", res)
	}
	if ns := ben.notifications(); ns[0]["pending"] == true {
		t.Error("answered request still pending")
	}
	if ns := ann.notifications(); len(ns) != 1 || ns[0]["kind"] != notifyFriendAccepted {
		t.Errorf("ann's bell: %v", kinds(ns))
	}

	if code, _ := ann.do("POST", "/api/messages/ben", `{"text":"  good game  "}`); code != 200 {
		t.Fatalf("message: %d", code)
	}
	_, res = ben.do("GET", "/api/notifications", "")
	if res["unreadChats"] != 1.0 {
		t.Errorf("unread chats %v", res["unreadChats"])
	}
	_, res = ben.do("GET", "/api/messages", "")
	conv := res["list"].([]any)[0].(map[string]any)
	if conv["with"].(map[string]any)["username"] != "ann" || conv["unread"] != 1.0 || conv["last"].(map[string]any)["text"] != "good game" {
		t.Errorf("conversations: %v", conv)
	}
	last := conv["last"].(map[string]any)["id"].(float64)
	ben.do("POST", "/api/messages/ann/read", `{"upTo":`+jsonNumber(last)+`}`)
	if _, res := ben.do("GET", "/api/notifications", ""); res["unreadChats"] != 0.0 {
		t.Errorf("unread after reading %v", res["unreadChats"])
	}

	// Privacy.
	cat.do("PUT", "/api/settings", `{"hideOnline":true,"offersFrom":"friends","requestsFrom":"nobody"}`)
	if code, _ := ann.do("POST", "/api/offers", `{"to":"cat","settings":{"time":{"base":300}}}`); code != 403 {
		t.Errorf("offer to cat: %d", code)
	}
	if code, _ := ann.do("POST", "/api/friends/cat", ""); code != 403 {
		t.Errorf("request to cat: %d", code)
	}
	if code, _ := cat.do("PUT", "/api/settings", `{"offersFrom":"strangers","requestsFrom":"nobody"}`); code != 400 {
		t.Errorf("bad privacy: %d", code)
	}

	// A match offer between friends shows up in the chat and the bell.
	if code, _ := ann.do("POST", "/api/offers", `{"to":"ben","settings":{"time":{"base":299}}}`); code != 400 {
		t.Errorf("offer off the presets: %d", code)
	}
	_, res = ann.do("POST", "/api/offers", `{"to":"ben","settings":{"fog":true,"time":{"base":300}}}`)
	offerID := jsonNumber(res["id"].(float64))
	_, res = ben.do("GET", "/api/messages/ann", "")
	msgs := res["messages"].([]any)
	offer := msgs[len(msgs)-1].(map[string]any)["offer"].(map[string]any)
	if offer["status"] != "open" || offer["from"] != "ann" {
		t.Errorf("offer in chat: %v", offer)
	}
	if res["friend"] != "friends" || res["with"].(map[string]any)["username"] != "ann" {
		t.Errorf("chat header: %v", res)
	}
	if code, _ := ann.do("POST", "/api/offers/"+offerID+"/accept", ""); code != 404 {
		t.Errorf("accepting your own offer: %d", code)
	}
	_, res = ben.do("POST", "/api/offers/"+offerID+"/accept", "")
	matchID, _ := res["matchId"].(string)
	m := site.hub.Get(matchID)
	if m == nil {
		t.Fatalf("accept: %v", res)
	}
	if code, _ := ben.do("POST", "/api/offers/"+offerID+"/decline", ""); code != 409 {
		t.Errorf("answering twice: %d", code)
	}
	if ns := ann.notifications(); ns[0]["kind"] != notifyOfferAccepted || ns[0]["matchId"] != matchID {
		t.Errorf("ann's bell after accept: %v", ns[0])
	}

	// The clock waits for both players, and a player who isn't there hears how it ended.
	m.mu.Lock()
	if m.users[white] == 0 || m.users[black] == 0 || m.running() || !m.settings.Fog {
		t.Errorf("offered match: users %v running %v", m.users, m.running())
	}
	key := func(c int) string { return m.seats[c] }
	m.mu.Unlock()
	cw, cb := newClient(key(white)), newClient(key(black))
	m.join(cw, &player{key: key(white), userID: m.users[white]})
	m.mu.Lock()
	running := m.running()
	m.mu.Unlock()
	if running {
		t.Error("clock started with one player there")
	}
	m.join(cb, &player{key: key(black), userID: m.users[black]})
	m.mu.Lock()
	running = m.running()
	m.mu.Unlock()
	if !running {
		t.Error("clock didn't start with both players there")
	}
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	m.leave(cw)
	m.handle(cb, inMsg{Type: "resign"})
	whiteName := "ann"
	if m.users[white] == ben.id(t) {
		whiteName = "ben"
	}
	whiteUser, blackUser := ann, ben
	if whiteName == "ben" {
		whiteUser, blackUser = ben, ann
	}
	if ns := whiteUser.notifications(); ns[0]["kind"] != notifyGameOver || ns[0]["data"].(map[string]any)["result"] != "won" {
		t.Errorf("white's bell after the game: %v", ns[0])
	}
	if ns := blackUser.notifications(); ns[0]["kind"] == notifyGameOver {
		t.Error("black was at the board but got a result notification")
	}

	// Profiles: the score between friends.
	_, res = ann.do("GET", "/api/users/ben", "")
	if res["friend"] != "friends" || res["canOffer"] != true || res["scoreWith"] == nil || res["stats"] == nil {
		t.Errorf("profile: %v", res)
	}

	// Deleting a chat takes both.
	if _, res := ann.do("POST", "/api/messages/ben/delete", `{"want":true}`); res["deleted"] != false {
		t.Errorf("one side deleted the chat: %v", res)
	}
	_, res = ben.do("GET", "/api/messages/ann", "")
	if res["deletion"].(map[string]any)["theirs"] != true {
		t.Errorf("deletion: %v", res["deletion"])
	}
	if _, res := ben.do("POST", "/api/messages/ann/delete", `{"want":true}`); res["deleted"] != true {
		t.Errorf("both sides didn't delete the chat: %v", res)
	}
	if _, res := ann.do("GET", "/api/messages/ben", ""); len(res["messages"].([]any)) != 0 {
		t.Errorf("messages left: %v", res["messages"])
	}

	ann.do("DELETE", "/api/friends/ben", "")
	if _, res := ben.do("GET", "/api/friends", ""); len(res["friends"].([]any)) != 0 {
		t.Errorf("still friends: %v", res)
	}
}

func (u *testUser) id(t *testing.T) int64 {
	got, err := u.site.store.UserByName(u.name)
	if err != nil || got == nil {
		t.Fatal(err)
	}
	return got.ID
}

func jsonNumber(f float64) string {
	b, _ := json.Marshal(int64(f))
	return string(b)
}

func TestSlowMoveNotifies(t *testing.T) {
	site := newTestSite(t)
	ann, ben := site.signup("ann"), site.signup("ben")
	ann.do("POST", "/api/friends/ben", "")
	ben.do("POST", "/api/friends/ann", "")
	_, res := ann.do("POST", "/api/offers", `{"to":"ben","settings":{},"color":"b"}`)
	_, res = ben.do("POST", "/api/offers/"+jsonNumber(res["id"].(float64))+"/accept", "")
	m := site.hub.Get(res["matchId"].(string))
	m.mu.Lock()
	seats, users := m.seats, m.users
	m.mu.Unlock()
	if users[black] != ann.id(t) {
		t.Fatalf("ann asked for black, got seats %v", users)
	}
	cw := newClient(seats[white])
	m.join(cw, &player{key: seats[white], userID: users[white]})

	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"}) // quick: no notification
	cb := newClient(seats[black])
	m.join(cb, &player{key: seats[black], userID: users[black]})
	m.handle(cb, inMsg{Type: "move", UCI: "e7e5"})
	m.leave(cb)
	m.mu.Lock()
	m.turnBegan = time.Now().Add(-5 * time.Hour)
	m.mu.Unlock()
	m.handle(cw, inMsg{Type: "move", UCI: "g1f3"})

	got := kinds(ann.notifications())
	moves := 0
	for _, k := range got {
		if k == notifyMove {
			moves++
		}
	}
	if moves != 1 {
		t.Errorf("black's bell %v, want one move notification", got)
	}
}

func TestLiveEvents(t *testing.T) {
	site := newTestSite(t)
	ann, ben := site.signup("ann"), site.signup("ben")
	ann.do("POST", "/api/friends/ben", "")
	ben.do("POST", "/api/friends/ann", "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", site.srv.URL+"/api/live", nil)
	res, err := ben.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type %q", res.Header.Get("Content-Type"))
	}
	lines := bufio.NewScanner(res.Body)
	lines.Scan() // retry
	// Ben is online now.
	time.Sleep(50 * time.Millisecond)
	if _, p := ann.do("GET", "/api/users/ben", ""); p["online"] != true {
		t.Errorf("ben not online: %v", p["online"])
	}
	ann.do("POST", "/api/messages/ben", `{"text":"there?"}`)
	for lines.Scan() {
		if line, ok := strings.CutPrefix(lines.Text(), "data: "); ok {
			var ev liveEvent
			json.Unmarshal([]byte(line), &ev)
			if ev.Type == "message" && ev.User == "ann" {
				return
			}
		}
	}
	t.Error("no message event")
}

func TestAccountChanges(t *testing.T) {
	site := newTestSite(t)
	ann := site.signup("ann")
	if code, _ := ann.do("POST", "/api/password", `{"current":"wrong one","new":"brand new pw"}`); code != 401 {
		t.Errorf("wrong current password: %d", code)
	}
	if code, _ := ann.do("POST", "/api/password", `{"current":"long enough","new":"brand new pw"}`); code != 200 {
		t.Errorf("change password: %d", code)
	}
	if _, me := ann.do("GET", "/api/settings", ""); me["offersFrom"] != "anyone" {
		t.Errorf("signed out by a password change: %v", me)
	}
	// A game of theirs stays, played by a guest.
	m := site.hub.Create(&player{key: "u:x", userID: ann.id(t), name: "ann"}, Settings{})
	for i := 0; ; i++ {
		if rec, _ := site.store.LatestGame(m.id); rec != nil {
			break
		}
		if i == 100 {
			t.Fatal("game never saved")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if code, _ := ann.do("POST", "/api/account/delete", `{"password":"brand new pw"}`); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := ann.do("GET", "/api/settings", ""); code != 401 {
		t.Errorf("still signed in after deleting: %d", code)
	}
	if rec, err := site.store.LatestGame(m.id); err != nil || rec == nil || rec.Users[white] != 0 {
		t.Errorf("game after deleting the account: %+v, %v", rec, err)
	}
}
