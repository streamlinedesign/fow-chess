package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie = "session"
	sessionTTL    = 90 * 24 * time.Hour
	// bcrypt only looks at the first 72 bytes, so longer passwords are refused rather than silently cut.
	minPassword = 8
	maxPassword = 72
)

var usernameRE = regexp.MustCompile(`^[A-Za-z0-9_-]{3,20}$`)

// Auth handles accounts and sessions.
type Auth struct {
	store   *Store
	limiter *failLimiter
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// meResponse is what the web app knows about the signed-in user.
type meResponse struct {
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

func (a *Auth) routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/signup", a.signup)
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("POST /api/password", a.changePassword)
	mux.HandleFunc("POST /api/logout-everywhere", a.logoutEverywhere)
	mux.HandleFunc("POST /api/account/delete", a.deleteAccount)
	// A guest gets null rather than an error, since being a guest is normal.
	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		u := a.user(r)
		if u == nil {
			writeJSON(w, nil)
			return
		}
		writeJSON(w, meResponse{u.Username, int(u.Rating + 0.5)})
	})
}

func (a *Auth) signup(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !readJSON(w, r, &c) {
		return
	}
	if !usernameRE.MatchString(c.Username) {
		jsonError(w, "Usernames are 3 to 20 letters, numbers, - or _.", http.StatusBadRequest)
		return
	}
	if len(c.Password) < minPassword || len(c.Password) > maxPassword {
		jsonError(w, "Passwords are 8 to 72 characters.", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "Couldn't create the account.", http.StatusInternalServerError)
		return
	}
	u, err := a.store.CreateUser(c.Username, hash)
	if err == ErrUsernameTaken {
		jsonError(w, "That username is taken.", http.StatusConflict)
		return
	}
	if err != nil {
		jsonError(w, "Couldn't create the account.", http.StatusInternalServerError)
		return
	}
	a.startSession(w, r, u)
}

func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !readJSON(w, r, &c) {
		return
	}
	key := clientIP(r)
	if !a.limiter.allow(key) {
		jsonError(w, "Too many attempts. Try again in a few minutes.", http.StatusTooManyRequests)
		return
	}
	u, err := a.store.UserByName(c.Username)
	if err != nil {
		jsonError(w, "Couldn't sign in.", http.StatusInternalServerError)
		return
	}
	// Compare against a dummy hash for unknown users so both cases take as long.
	hash := dummyHash
	if u != nil {
		hash = u.hash
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(c.Password)) != nil || u == nil {
		a.limiter.fail(key)
		jsonError(w, "Wrong username or password.", http.StatusUnauthorized)
		return
	}
	a.startSession(w, r, u)
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.store.DeleteSession(hashToken(c.Value))
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// checkPassword is for changes to a signed-in account: it answers and returns
// nil unless the request is signed in and password is theirs.
func (a *Auth) checkPassword(w http.ResponseWriter, r *http.Request, password string) *User {
	u := a.user(r)
	if u == nil {
		jsonError(w, "Log in first.", http.StatusUnauthorized)
		return nil
	}
	key := clientIP(r)
	if !a.limiter.allow(key) {
		jsonError(w, "Too many attempts. Try again in a few minutes.", http.StatusTooManyRequests)
		return nil
	}
	if bcrypt.CompareHashAndPassword(u.hash, []byte(password)) != nil {
		a.limiter.fail(key)
		jsonError(w, "That's not your current password.", http.StatusUnauthorized)
		return nil
	}
	return u
}

// changePassword sets a new password and signs out every other device.
func (a *Auth) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	u := a.checkPassword(w, r, body.Current)
	if u == nil {
		return
	}
	if len(body.New) < minPassword || len(body.New) > maxPassword {
		jsonError(w, "Passwords are 8 to 72 characters.", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.New), bcrypt.DefaultCost)
	if err == nil {
		err = a.store.SetPassword(u.ID, hash)
	}
	if err == nil {
		err = a.store.DeleteSessions(u.ID)
	}
	if err != nil {
		jsonError(w, "Couldn't change the password.", http.StatusInternalServerError)
		return
	}
	a.startSession(w, r, u)
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

// logoutEverywhere ends every session of the signed-in account, this one too.
func (a *Auth) logoutEverywhere(w http.ResponseWriter, r *http.Request) {
	if u := a.user(r); u != nil {
		if err := a.store.DeleteSessions(u.ID); err != nil {
			jsonError(w, "Couldn't log out.", http.StatusInternalServerError)
			return
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// deleteAccount deletes the signed-in account once its password is confirmed.
func (a *Auth) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	u := a.checkPassword(w, r, body.Password)
	if u == nil {
		return
	}
	if err := a.store.DeleteUser(u.ID); err != nil {
		jsonError(w, "Couldn't delete the account.", http.StatusInternalServerError)
		return
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *Auth) startSession(w http.ResponseWriter, r *http.Request, u *User) {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.RawURLEncoding.EncodeToString(b)
	expires := time.Now().Add(sessionTTL)
	// Only a hash is stored, so a leaked database can't be used to sign in.
	if err := a.store.CreateSession(hashToken(token), u.ID, expires); err != nil {
		jsonError(w, "Couldn't sign in.", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
	})
	writeJSON(w, meResponse{u.Username, int(u.Rating + 0.5)})
}

// user returns the signed-in user for a request, or nil for a guest.
func (a *Auth) user(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	u, _ := a.store.SessionUser(hashToken(c.Value))
	return u
}

// userForSocket is user for a websocket upgrade, which browsers send with
// cookies even from other sites. A cookie only counts when the page asking is
// our own; any other origin plays as a guest.
func (a *Auth) userForSocket(r *http.Request) *User {
	if o := r.Header.Get("Origin"); o != "" {
		ou, err := url.Parse(o)
		if err != nil || ou.Host != r.Host {
			return nil
		}
	}
	return a.user(r)
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not a real password"), bcrypt.DefaultCost)

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// failLimiter allows a handful of failed logins per address before making it wait.
type failLimiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

const (
	maxFails   = 10
	failWindow = 15 * time.Minute
)

func newFailLimiter() *failLimiter { return &failLimiter{fails: map[string][]time.Time{}} }

func (l *failLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key)) < maxFails
}

func (l *failLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fails[key] = append(l.recent(key), time.Now())
}

// recent drops failures older than the window. Callers hold l.mu.
func (l *failLimiter) recent(key string) []time.Time {
	ts := l.fails[key]
	i := 0
	for i < len(ts) && time.Since(ts[i]) > failWindow {
		i++
	}
	if i == len(ts) {
		delete(l.fails, key)
		return nil
	}
	l.fails[key] = ts[i:]
	return ts[i:]
}

// readJSON decodes a JSON body. Requiring the JSON content type means another
// site can't submit these forms: browsers only send it cross-site after a CORS
// preflight, which this server never approves.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		jsonError(w, "Bad request.", http.StatusUnsupportedMediaType)
		return false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(v); err != nil {
		jsonError(w, "Bad request.", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// errorCodes names each error message, so the client can show it in the
// user's language. The English text is sent too, as a fallback.
var errorCodes = map[string]string{
	"Couldn't load the profile.":                      "load_profile",
	"No such player.":                                 "no_player",
	"Too many attempts. Try again in a few minutes.":  "too_many",
	"Those match settings aren't available.":          "bad_settings",
	"Something went wrong.":                           "server",
	"Passwords are 8 to 72 characters.":               "password_length",
	"No such game.":                                   "no_game",
	"Log in first.":                                   "log_in",
	"Couldn't sign in.":                               "sign_in_failed",
	"Couldn't load the leaderboard.":                  "load_leaderboard",
	"Couldn't load the game.":                         "load_game",
	"Couldn't create the account.":                    "create_account",
	"Wrong username or password.":                     "wrong_login",
	"Write something first.":                          "empty_message",
	"Usernames are 3 to 20 letters, numbers, - or _.": "username_rules",
	"Those settings aren't allowed.":                  "settings_not_allowed",
	"This offer has already been answered.":           "offer_answered",
	"This game is still being played.":                "game_in_progress",
	"That's you.":                                     "thats_you",
	"That's not your current password.":               "wrong_password",
	"That username is taken.":                         "username_taken",
	"That player has left.":                           "player_left",
	"Pick white, black or random.":                    "pick_colour",
	"Only friends can message each other.":            "friends_only",
	"No such offer.":                                  "no_offer",
	"Messages are up to 1000 characters.":             "message_length",
	"Couldn't log out.":                               "log_out",
	"Couldn't delete the account.":                    "delete_account",
	"Couldn't change the password.":                   "change_password",
	"Bad request.":                                    "bad_request",
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	writeError(w, status, map[string]string{"error": msg, "code": errorCodes[msg]})
}

// jsonNamedError sends an error about a player, whose name the client puts into its own text.
func jsonNamedError(w http.ResponseWriter, msg, code, name string, status int) {
	writeError(w, status, map[string]string{"error": msg, "code": code, "name": name})
}

func writeError(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func jsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func parseJSON(s string, v any) {
	json.Unmarshal([]byte(s), v)
}
