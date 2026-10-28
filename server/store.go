package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store keeps users, sessions and games in Postgres, so games survive a restart.
// The tables are defined in migrations/.
type Store struct {
	db     *sql.DB
	writes chan gameRecord
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

// defaultDatabaseURL is a database called chess in the Postgres installed on
// this machine, reached through its local socket as the current OS user. That's
// how Homebrew and Postgres.app set Postgres up, so it works with no config.
const defaultDatabaseURL = "postgres:///chess"

// guestGameTTL is how long games without a signed-in player are kept.
const guestGameTTL = 30 * 24 * time.Hour

// OpenStore connects to Postgres and brings its tables up to date.
// The database itself is created if it doesn't exist yet.
func OpenStore(url string) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := createDatabaseIfMissing(ctx, url); err != nil {
		return nil, fmt.Errorf("can't reach Postgres at %s (is it running?): %w", redactURL(url), err)
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db, writes: make(chan gameRecord, 1024), done: make(chan struct{})}
	go s.writer()
	go s.cleanup()
	return s, nil
}

// createDatabaseIfMissing connects to the database in url and, if Postgres says
// it doesn't exist, creates it from the built-in postgres database.
func createDatabaseIfMissing(ctx context.Context, url string) error {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return err
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err == nil {
		return conn.Close(ctx)
	}
	if pgErr := (*pgconn.PgError)(nil); !errors.As(err, &pgErr) || pgErr.Code != "3D000" { // invalid_catalog_name
		return err
	}
	name := cfg.Database
	cfg.Database = "postgres"
	if conn, err = pgx.ConnectConfig(ctx, cfg); err != nil {
		return err
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()); err != nil {
		return fmt.Errorf("creating database %s: %w", name, err)
	}
	log.Printf("created database %s", name)
	return nil
}

// Close finishes queued game writes and closes the database.
func (s *Store) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	close(s.writes)
	<-s.done
	s.db.Close()
}

// gameRecord is one game's row as of a moment in time.
type gameRecord struct {
	MatchID  string
	Round    int
	Settings Settings
	Seats    [2]string
	Users    [2]int64 // 0 for a guest or an empty seat
	Moves    string
	// StartFEN is the starting position of a game from a setup mode; empty for the usual one.
	StartFEN  string
	Result    string
	Reason    string
	CreatedAt time.Time
	UpdatedAt time.Time
	// Clock is each side's time left as of TurnStart, when the side to move's
	// clock started. A zero TurnStart means no clock is running.
	Clock     [2]time.Duration
	TurnStart time.Time
	// Rated games move both players' ratings once they finish.
	Rated      bool
	RatingDiff [2]*int // nil until the ratings have been updated
	whiteName  string
	blackName  string
	// onRated, if set, is called with the rating changes once they're saved.
	onRated func(diff [2]int)
}

// sniper is whether the game was a Sniper Win; see sniperWin.
func (r *gameRecord) sniper() bool {
	return sniperWin(r.Result, r.Reason, len(strings.Fields(r.Moves)))
}

// deadline is when the side to move runs out of time, or nil if no clock is running.
func (r *gameRecord) deadline() *time.Time {
	if r.TurnStart.IsZero() {
		return nil
	}
	d := r.TurnStart.Add(r.Clock[len(strings.Fields(r.Moves))%2])
	return &d
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
		var turnStart any
		if !r.TurnStart.IsZero() {
			turnStart = r.TurnStart
		}
		_, err := s.db.Exec(`
			INSERT INTO games (match_id, round, settings, white, black, white_user, black_user, moves, result, reason, created_at, updated_at,
				white_clock_ms, black_clock_ms, turn_started_at, deadline, rated, start_fen)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
			ON CONFLICT (match_id, round) DO UPDATE SET
				white = excluded.white, black = excluded.black,
				white_user = excluded.white_user, black_user = excluded.black_user,
				moves = excluded.moves, result = excluded.result, reason = excluded.reason,
				updated_at = excluded.updated_at,
				white_clock_ms = excluded.white_clock_ms, black_clock_ms = excluded.black_clock_ms,
				turn_started_at = excluded.turn_started_at, deadline = excluded.deadline, rated = excluded.rated,
				start_fen = excluded.start_fen`,
			r.MatchID, r.Round, settings, r.Seats[white], r.Seats[black], nullID(r.Users[white]), nullID(r.Users[black]),
			r.Moves, r.Result, r.Reason, r.CreatedAt, r.UpdatedAt,
			r.Clock[white].Milliseconds(), r.Clock[black].Milliseconds(), turnStart, r.deadline(), r.Rated, r.StartFEN)
		if err != nil {
			log.Printf("saving game %s/%d: %v", r.MatchID, r.Round, err)
			continue
		}
		if r.Rated && r.Result != "" {
			diff, err := s.rate(r)
			if err != nil {
				log.Printf("rating game %s/%d: %v", r.MatchID, r.Round, err)
			} else if diff != nil && r.onRated != nil {
				r.onRated(*diff)
			}
		}
	}
}

// rate updates both players' ratings for a finished game and records the
// changes on the game. It returns nil if the game was already rated.
func (s *Store) rate(r gameRecord) (*[2]int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var done bool
	err = tx.QueryRow(`SELECT white_rating_diff IS NOT NULL FROM games WHERE match_id = $1 AND round = $2 FOR UPDATE`,
		r.MatchID, r.Round).Scan(&done)
	if err != nil || done {
		return nil, err
	}
	// Lock in id order so two games finishing at once can't deadlock.
	rows, err := tx.Query(`SELECT id, rating, deviation, volatility FROM users WHERE id IN ($1, $2) ORDER BY id FOR UPDATE`,
		r.Users[white], r.Users[black])
	if err != nil {
		return nil, err
	}
	var before [2]glicko
	for rows.Next() {
		var id int64
		var g glicko
		if err := rows.Scan(&id, &g.Rating, &g.Deviation, &g.Volatility); err != nil {
			rows.Close()
			return nil, err
		}
		for c := range r.Users {
			if r.Users[c] == id {
				before[c] = g
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	score := map[string]float64{"1-0": 1, "0-1": 0, "1/2-1/2": 0.5}[r.Result]
	after := [2]glicko{
		before[white].rate([]glickoResult{{before[black], score}}),
		before[black].rate([]glickoResult{{before[white], 1 - score}}),
	}
	// A Sniper Win moves the ratings only part of the way a whole game would.
	if r.sniper() {
		for c := range after {
			after[c] = before[c].toward(after[c], sniperWeight)
		}
	}
	var diff [2]int
	for c := range after {
		diff[c] = int(math.Round(after[c].Rating)) - int(math.Round(before[c].Rating))
		if _, err := tx.Exec(`UPDATE users SET rating = $2, deviation = $3, volatility = $4, rated_games = rated_games + 1 WHERE id = $1`,
			r.Users[c], after[c].Rating, after[c].Deviation, after[c].Volatility); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(`UPDATE games SET white_rating_diff = $3, black_rating_diff = $4 WHERE match_id = $1 AND round = $2`,
		r.MatchID, r.Round, diff[white], diff[black]); err != nil {
		return nil, err
	}
	return &diff, tx.Commit()
}

// ExpiredGames lists games still on whose side to move has run out of time.
func (s *Store) ExpiredGames() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT match_id FROM games WHERE deadline < now() AND result = ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// leader is one row of the leaderboard.
type leader struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	Games    int    `json:"games"`
}

// Leaderboard lists the best-rated players who have played a rated game.
func (s *Store) Leaderboard(limit int) ([]leader, error) {
	rows, err := s.db.Query(`SELECT username, rating, rated_games FROM users WHERE rated_games > 0 ORDER BY rating DESC, id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []leader{}
	for rows.Next() {
		var l leader
		var rating float64
		if err := rows.Scan(&l.Username, &rating, &l.Games); err != nil {
			return nil, err
		}
		l.Rank, l.Rating = len(out)+1, int(math.Round(rating))
		out = append(out, l)
	}
	return out, rows.Err()
}

// Rank is a user's place on the leaderboard, or nil if they haven't played a rated game.
func (s *Store) Rank(userID int64) (*leader, error) {
	var l leader
	var rating float64
	err := s.db.QueryRow(`
		SELECT u.username, u.rating, u.rated_games,
			(SELECT count(*) FROM users o WHERE o.rated_games > 0 AND (o.rating > u.rating OR (o.rating = u.rating AND o.id < u.id))) + 1
		FROM users u WHERE u.id = $1 AND u.rated_games > 0`, userID).Scan(&l.Username, &rating, &l.Games, &l.Rank)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l.Rating = int(math.Round(rating))
	return &l, nil
}

// cleanup deletes old guest games and expired sessions once a day.
func (s *Store) cleanup() {
	for {
		s.db.Exec(`DELETE FROM games WHERE white_user IS NULL AND black_user IS NULL AND updated_at < $1`, time.Now().Add(-guestGameTTL))
		s.db.Exec(`DELETE FROM sessions WHERE expires_at < now()`)
		select {
		case <-s.done:
			return
		case <-time.After(24 * time.Hour):
		}
	}
}

// LatestGame loads the newest round of a match, or nil if there is none.
func (s *Store) LatestGame(matchID string) (*gameRecord, error) {
	rows, err := s.queryGames(`WHERE g.match_id = $1 ORDER BY g.round DESC LIMIT 1`, matchID)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// Game loads one round of a match, or nil if it doesn't exist.
func (s *Store) Game(matchID string, round int) (*gameRecord, error) {
	rows, err := s.queryGames(`WHERE g.match_id = $1 AND g.round = $2`, matchID, round)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// GamesOf lists a user's games, newest first.
func (s *Store) GamesOf(userID int64, limit int) ([]gameRecord, error) {
	return s.queryGames(`WHERE g.white_user = $1 OR g.black_user = $1 ORDER BY g.updated_at DESC LIMIT $2`, userID, limit)
}

func (s *Store) queryGames(where string, args ...any) ([]gameRecord, error) {
	rows, err := s.db.Query(`
		SELECT g.match_id, g.round, g.settings::text, g.white, g.black, coalesce(g.white_user, 0), coalesce(g.black_user, 0),
			g.moves, g.result, g.reason, g.created_at, g.updated_at, coalesce(wu.username, ''), coalesce(bu.username, ''),
			coalesce(g.white_clock_ms, 0), coalesce(g.black_clock_ms, 0), g.turn_started_at, g.rated, g.white_rating_diff, g.black_rating_diff,
			g.start_fen
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
		var clock [2]int64
		var turnStart sql.NullTime
		var diff [2]sql.NullInt64
		if err := rows.Scan(&r.MatchID, &r.Round, &settings, &r.Seats[white], &r.Seats[black], &r.Users[white], &r.Users[black],
			&r.Moves, &r.Result, &r.Reason, &r.CreatedAt, &r.UpdatedAt, &r.whiteName, &r.blackName,
			&clock[white], &clock[black], &turnStart, &r.Rated, &diff[white], &diff[black], &r.StartFEN); err != nil {
			return nil, err
		}
		for c := range clock {
			r.Clock[c] = time.Duration(clock[c]) * time.Millisecond
			if diff[c].Valid {
				d := int(diff[c].Int64)
				r.RatingDiff[c] = &d
			}
		}
		r.TurnStart = turnStart.Time
		parseJSON(settings, &r.Settings)
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
	var id int64
	err := s.db.QueryRow(`INSERT INTO users (username, password_hash, created_at) VALUES ($1, $2, $3) RETURNING id`, username, hash, now).Scan(&id)
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return nil, ErrUsernameTaken
	}
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Username: username, Rating: 1500, CreatedAt: now}, nil
}

// UserByName finds a user regardless of case, or returns nil.
func (s *Store) UserByName(username string) (*User, error) {
	return s.user(`lower(username) = lower($1)`, username)
}

func (s *Store) UserByID(id int64) (*User, error) {
	return s.user(`id = $1`, id)
}

func (s *Store) user(where string, arg any) (*User, error) {
	var u User
	err := s.db.QueryRow(`SELECT id, username, rating, created_at, password_hash FROM users WHERE `+where, arg).
		Scan(&u.ID, &u.Username, &u.Rating, &u.CreatedAt, &u.hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CreateSession(tokenHash []byte, userID int64, expires time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, tokenHash, userID, expires)
	return err
}

// SessionUser returns the user a session token belongs to, or nil if it's unknown or expired.
func (s *Store) SessionUser(tokenHash []byte) (*User, error) {
	var id int64
	err := s.db.QueryRow(`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`, tokenHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.UserByID(id)
}

func (s *Store) DeleteSession(tokenHash []byte) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}
