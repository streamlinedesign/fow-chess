package main

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store keeps users, sessions and games in one SQLite file, so the server stays a
// single binary and games survive a restart.
type Store struct {
	db     *sql.DB
	writes chan gameRecord
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY,
	username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
	password_hash BLOB NOT NULL,
	-- Glicko-2, one rating across every game mode.
	rating        REAL NOT NULL DEFAULT 1500,
	deviation     REAL NOT NULL DEFAULT 350,
	volatility    REAL NOT NULL DEFAULT 0.06,
	created_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash BLOB PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at INTEGER NOT NULL
);

-- One row per game: a match's rematches are rounds 1, 2, 3...
CREATE TABLE IF NOT EXISTS games (
	match_id     TEXT NOT NULL,
	round        INTEGER NOT NULL,
	settings     TEXT NOT NULL,
	white        TEXT NOT NULL DEFAULT '', -- seat key: "u:<id>" or a guest browser ID
	black        TEXT NOT NULL DEFAULT '',
	white_user   INTEGER REFERENCES users(id),
	black_user   INTEGER REFERENCES users(id),
	moves        TEXT NOT NULL DEFAULT '', -- UCI, space separated
	result       TEXT NOT NULL DEFAULT '', -- empty while the game is on
	reason       TEXT NOT NULL DEFAULT '',
	created_at   INTEGER NOT NULL,
	updated_at   INTEGER NOT NULL,
	PRIMARY KEY (match_id, round)
);
CREATE INDEX IF NOT EXISTS games_white_user ON games(white_user, updated_at);
CREATE INDEX IF NOT EXISTS games_black_user ON games(black_user, updated_at);
`

// guestGameTTL is how long games without a signed-in player are kept.
const guestGameTTL = 30 * 24 * time.Hour

func OpenStore(path string) (*Store, error) {
	// WAL lets reads run alongside the single writer; busy_timeout rides out brief lock waits.
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db, writes: make(chan gameRecord, 1024), done: make(chan struct{})}
	go s.writer()
	go s.cleanup()
	return s, nil
}

// Close finishes queued game writes and closes the database.
func (s *Store) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	close(s.writes)
	<-s.done
	s.db.Close()
}

// gameRecord is one game's row as of a moment in time.
type gameRecord struct {
	MatchID   string
	Round     int
	Settings  Settings
	Seats     [2]string
	Users     [2]int64 // 0 for a guest or an empty seat
	Moves     string
	Result    string
	Reason    string
	CreatedAt time.Time
	UpdatedAt time.Time
	whiteName string
	blackName string
}

// SaveGame queues a game snapshot. Writes happen off the match's lock so a
// move is never held up by the disk.
func (s *Store) SaveGame(r gameRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		log.Printf("not saving game %s/%d: shutting down", r.MatchID, r.Round)
		return
	}
	s.writes <- r
}

func (s *Store) writer() {
	defer close(s.done)
	for r := range s.writes {
		settings, _ := jsonString(r.Settings)
		_, err := s.db.Exec(`
			INSERT INTO games (match_id, round, settings, white, black, white_user, black_user, moves, result, reason, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (match_id, round) DO UPDATE SET
				white = excluded.white, black = excluded.black,
				white_user = excluded.white_user, black_user = excluded.black_user,
				moves = excluded.moves, result = excluded.result, reason = excluded.reason,
				updated_at = excluded.updated_at`,
			r.MatchID, r.Round, settings, r.Seats[white], r.Seats[black], nullID(r.Users[white]), nullID(r.Users[black]),
			r.Moves, r.Result, r.Reason, r.CreatedAt.Unix(), r.UpdatedAt.Unix())
		if err != nil {
			log.Printf("saving game %s/%d: %v", r.MatchID, r.Round, err)
		}
	}
}

// cleanup deletes old guest games and expired sessions once a day.
func (s *Store) cleanup() {
	for {
		cutoff := time.Now().Add(-guestGameTTL).Unix()
		s.db.Exec(`DELETE FROM games WHERE white_user IS NULL AND black_user IS NULL AND updated_at < ?`, cutoff)
		s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().Unix())
		select {
		case <-s.done:
			return
		case <-time.After(24 * time.Hour):
		}
	}
}

// LatestGame loads the newest round of a match, or nil if there is none.
func (s *Store) LatestGame(matchID string) (*gameRecord, error) {
	rows, err := s.queryGames(`WHERE g.match_id = ? ORDER BY g.round DESC LIMIT 1`, matchID)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// Game loads one round of a match, or nil if it doesn't exist.
func (s *Store) Game(matchID string, round int) (*gameRecord, error) {
	rows, err := s.queryGames(`WHERE g.match_id = ? AND g.round = ?`, matchID, round)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// GamesOf lists a user's games, newest first.
func (s *Store) GamesOf(userID int64, limit int) ([]gameRecord, error) {
	return s.queryGames(`WHERE g.white_user = ?1 OR g.black_user = ?1 ORDER BY g.updated_at DESC LIMIT ?2`, userID, limit)
}

func (s *Store) queryGames(where string, args ...any) ([]gameRecord, error) {
	rows, err := s.db.Query(`
		SELECT g.match_id, g.round, g.settings, g.white, g.black, coalesce(g.white_user, 0), coalesce(g.black_user, 0),
			g.moves, g.result, g.reason, g.created_at, g.updated_at, coalesce(wu.username, ''), coalesce(bu.username, '')
		FROM games g
		LEFT JOIN users wu ON wu.id = g.white_user
		LEFT JOIN users bu ON bu.id = g.black_user
		`+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gameRecord
	for rows.Next() {
		var r gameRecord
		var settings string
		var created, updated int64
		if err := rows.Scan(&r.MatchID, &r.Round, &settings, &r.Seats[white], &r.Seats[black], &r.Users[white], &r.Users[black],
			&r.Moves, &r.Result, &r.Reason, &created, &updated, &r.whiteName, &r.blackName); err != nil {
			return nil, err
		}
		parseJSON(settings, &r.Settings)
		r.CreatedAt, r.UpdatedAt = time.Unix(created, 0), time.Unix(updated, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// User is a signed-in player.
type User struct {
	ID        int64
	Username  string
	Rating    float64
	CreatedAt time.Time
	hash      []byte
}

var ErrUsernameTaken = errors.New("username taken")

func (s *Store) CreateUser(username string, hash []byte) (*User, error) {
	now := time.Now()
	res, err := s.db.Exec(`INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`, username, hash, now.Unix())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, Rating: 1500, CreatedAt: now}, nil
}

// UserByName finds a user regardless of case, or returns nil.
func (s *Store) UserByName(username string) (*User, error) {
	return s.user(`username = ?`, username)
}

func (s *Store) UserByID(id int64) (*User, error) {
	return s.user(`id = ?`, id)
}

func (s *Store) user(where string, arg any) (*User, error) {
	var u User
	var created int64
	err := s.db.QueryRow(`SELECT id, username, rating, created_at, password_hash FROM users WHERE `+where, arg).
		Scan(&u.ID, &u.Username, &u.Rating, &created, &u.hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt = time.Unix(created, 0)
	return &u, nil
}

func (s *Store) CreateSession(tokenHash []byte, userID int64, expires time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`, tokenHash, userID, expires.Unix())
	return err
}

// SessionUser returns the user a session token belongs to, or nil if it's unknown or expired.
func (s *Store) SessionUser(tokenHash []byte) (*User, error) {
	var id int64
	err := s.db.QueryRow(`SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > ?`, tokenHash, time.Now().Unix()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.UserByID(id)
}

func (s *Store) DeleteSession(tokenHash []byte) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}
