package main

import (
	"time"
)

type tournamentRow struct {
	ID       int64     `json:"id"`
	Kind     string    `json:"kind"`
	StartsAt time.Time `json:"startsAt"`
	EndsAt   time.Time `json:"endsAt"`
	Status   string    `json:"status"`
	Players  int       `json:"players"`
}

type tournamentPlayer struct {
	UserID   int64  `json:"-"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	Score    int    `json:"score"`
	Games    int    `json:"games"`
	Wins     int    `json:"wins"`
	Paused   bool   `json:"paused,omitempty"`
}

type tournamentGame struct {
	MatchID string
	Round   int
	White   int64
	Black   int64
}

// EnsureTournament creates a scheduled tournament unless one already exists at that time.
func (s *Store) EnsureTournament(kind string, startsAt, endsAt time.Time) error {
	// The existence check comes first so a tick that adds nothing doesn't
	// burn an id; the unique key still guards the race.
	_, err := s.db.Exec(`INSERT INTO tournaments (kind, starts_at, ends_at)
		SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM tournaments WHERE kind = $1 AND starts_at = $2)
		ON CONFLICT (kind, starts_at) DO NOTHING`, kind, startsAt, endsAt)
	return err
}

const tournamentColumns = `t.id, t.kind, t.starts_at, t.ends_at, t.status,
	(SELECT count(*) FROM tournament_players p WHERE p.tournament_id = t.id)`

func (s *Store) tournaments(where string, args ...any) ([]tournamentRow, error) {
	rows, err := s.db.Query(`SELECT `+tournamentColumns+` FROM tournaments t `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []tournamentRow{}
	for rows.Next() {
		var t tournamentRow
		if err := rows.Scan(&t.ID, &t.Kind, &t.StartsAt, &t.EndsAt, &t.Status, &t.Players); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Tournament returns one tournament, or nil.
func (s *Store) Tournament(id int64) (*tournamentRow, error) {
	ts, err := s.tournaments(`WHERE t.id = $1`, id)
	if err != nil || len(ts) == 0 {
		return nil, err
	}
	return &ts[0], nil
}

// Upcoming lists tournaments that haven't finished, soonest first.
func (s *Store) UpcomingTournaments() ([]tournamentRow, error) {
	return s.tournaments(`WHERE t.status IN ('scheduled', 'running') ORDER BY t.starts_at`)
}

// RecentTournaments lists the newest finished tournaments.
func (s *Store) RecentTournaments(limit int) ([]tournamentRow, error) {
	return s.tournaments(`WHERE t.status = 'finished' ORDER BY t.starts_at DESC LIMIT $1`, limit)
}

func (s *Store) SetTournamentStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE tournaments SET status = $2 WHERE id = $1`, id, status)
	return err
}

// JoinTournament adds a player, or lets a paused one back in.
func (s *Store) JoinTournament(id, userID int64) error {
	_, err := s.db.Exec(`INSERT INTO tournament_players (tournament_id, user_id, joined_at) VALUES ($1, $2, now())
		ON CONFLICT (tournament_id, user_id) DO UPDATE SET paused = false`, id, userID)
	return err
}

func (s *Store) LeaveTournament(id, userID int64) error {
	_, err := s.db.Exec(`DELETE FROM tournament_players WHERE tournament_id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *Store) PauseTournament(id, userID int64, paused bool) error {
	_, err := s.db.Exec(`UPDATE tournament_players SET paused = $3 WHERE tournament_id = $1 AND user_id = $2`, id, userID, paused)
	return err
}

// TournamentPlayers lists a tournament's players, best first.
func (s *Store) TournamentPlayers(id int64) ([]tournamentPlayer, error) {
	rows, err := s.db.Query(`SELECT p.user_id, u.username, round(u.rating), p.score, p.games, p.wins, p.paused
		FROM tournament_players p JOIN users u ON u.id = p.user_id
		WHERE p.tournament_id = $1 ORDER BY p.score DESC, p.wins DESC, p.games, u.username`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []tournamentPlayer{}
	for rows.Next() {
		var p tournamentPlayer
		if err := rows.Scan(&p.UserID, &p.Username, &p.Rating, &p.Score, &p.Games, &p.Wins, &p.Paused); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// TournamentsOf lists the tournaments a player is in that haven't finished.
func (s *Store) TournamentsOf(userID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT p.tournament_id FROM tournament_players p JOIN tournaments t ON t.id = p.tournament_id
		WHERE p.user_id = $1 AND t.status IN ('scheduled', 'running')`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) AddTournamentGame(id int64, g tournamentGame) error {
	_, err := s.db.Exec(`INSERT INTO tournament_games (tournament_id, match_id, round, white_user, black_user, created_at)
		VALUES ($1, $2, $3, $4, $5, now())`, id, g.MatchID, g.Round, g.White, g.Black)
	return err
}

// ScoreTournamentGame records a result and the points it's worth, once.
func (s *Store) ScoreTournamentGame(id int64, g tournamentGame, result string, points [2]int, wins [2]bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE tournament_games SET result = $3 WHERE match_id = $1 AND round = $2 AND result = ''`, g.MatchID, g.Round, result)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	for i, user := range [2]int64{g.White, g.Black} {
		win := 0
		if wins[i] {
			win = 1
		}
		if _, err := tx.Exec(`UPDATE tournament_players SET score = score + $3, games = games + 1, wins = wins + $4
			WHERE tournament_id = $1 AND user_id = $2`, id, user, points[i], win); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// OpenTournamentGames lists a tournament's games still being played.
func (s *Store) OpenTournamentGames(id int64) ([]tournamentGame, error) {
	rows, err := s.db.Query(`SELECT match_id, round, white_user, black_user FROM tournament_games WHERE tournament_id = $1 AND result = ''`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tournamentGame
	for rows.Next() {
		var g tournamentGame
		if err := rows.Scan(&g.MatchID, &g.Round, &g.White, &g.Black); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
