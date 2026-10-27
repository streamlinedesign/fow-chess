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
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
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

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func parseJSON(s string, v any) {
	json.Unmarshal([]byte(s), v)
}
