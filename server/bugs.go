package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// version is the release this binary was built from; `make build` sets it
// from VERSION. Bug reports record it.
var version = "dev"

// bugCategories are what a bug report can be about, as the report dialog
// offers them.
var bugCategories = map[string]bool{
	"ui":         true, // how something looks or reacts
	"rules":      true, // a move, the fog, or a result that seems wrong
	"connection": true, // the game froze, lagged or lost the opponent
	"clock":      true,
	"bots":       true,
	"account":    true, // signing in, profiles, friends, messages
	"other":      true,
}

const maxBugBody = 2000

var matchCode = regexp.MustCompile(`^[A-Z0-9]{4,12}$`)

type bugReport struct {
	ID        int64     `json:"id"`
	Category  string    `json:"category"`
	MatchID   string    `json:"match"`
	Body      string    `json:"text"`
	Page      string    `json:"page"`
	Username  string    `json:"username,omitempty"`
	Player    string    `json:"player,omitempty"`
	UserAgent string    `json:"userAgent"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
}

// bugRoutes takes bug reports from the site. Anyone can send one, signed in
// or not, a few per hour.
func bugRoutes(mux *http.ServeMux, store *Store, auth *Auth) {
	limiter := newFailLimiter()
	mux.HandleFunc("POST /api/bugs", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Category string `json:"category"`
			Match    string `json:"match"`
			Text     string `json:"text"`
			Page     string `json:"page"`
			Player   string `json:"player"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
			jsonError(w, "Bad request.", http.StatusBadRequest)
			return
		}
		rep := bugReport{
			Category:  req.Category,
			MatchID:   strings.ToUpper(strings.TrimSpace(req.Match)),
			Body:      strings.TrimSpace(req.Text),
			Page:      clip(req.Page, 200),
			UserAgent: clip(r.UserAgent(), 300),
			Version:   version,
		}
		if !bugCategories[rep.Category] {
			jsonError(w, "Pick what the bug is about.", http.StatusBadRequest)
			return
		}
		if rep.MatchID != "" && !matchCode.MatchString(rep.MatchID) {
			jsonError(w, "That isn't a match code.", http.StatusBadRequest)
			return
		}
		if rep.Body == "" {
			jsonError(w, "Write what happened first.", http.StatusBadRequest)
			return
		}
		if utf8.RuneCountInString(rep.Body) > maxBugBody {
			jsonError(w, "Reports are up to 2000 characters.", http.StatusBadRequest)
			return
		}
		// Count per address, and per account too: a guest's browser ID is
		// whatever it says, so it only labels the report.
		var userID int64
		keys := []string{"ip:" + clientIP(r)}
		if u := auth.user(r); u != nil {
			userID, rep.Username = u.ID, u.Username
			keys = append(keys, fmt.Sprintf("user:%d", u.ID))
		} else if guestIDRE.MatchString(req.Player) {
			rep.Player = req.Player
		}
		for _, key := range keys {
			if !limiter.allow(key) {
				jsonError(w, "Too many reports. Try again later.", http.StatusTooManyRequests)
				return
			}
		}
		for _, key := range keys {
			limiter.fail(key)
		}
		if err := store.SaveBugReport(&rep, userID); err != nil {
			log.Printf("saving bug report: %v", err)
			jsonError(w, "Something went wrong.", http.StatusInternalServerError)
			return
		}
		log.Printf("bug report %d (%s, match %q)", rep.ID, rep.Category, rep.MatchID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{"id": rep.ID})
	})
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func (s *Store) SaveBugReport(r *bugReport, userID int64) error {
	return s.db.QueryRow(`INSERT INTO bug_reports (category, match_id, body, page, user_id, player, user_agent, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at`,
		r.Category, r.MatchID, r.Body, r.Page, nullID(userID), r.Player, r.UserAgent, r.Version,
	).Scan(&r.ID, &r.CreatedAt)
}

// BugReports returns the newest reports first.
func (s *Store) BugReports(limit int) ([]bugReport, error) {
	rows, err := s.db.Query(`SELECT b.id, b.category, b.match_id, b.body, b.page, coalesce(u.username, ''),
			b.player, b.user_agent, b.version, b.created_at
		FROM bug_reports b LEFT JOIN users u ON u.id = b.user_id
		ORDER BY b.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bugReport
	for rows.Next() {
		var r bugReport
		if err := rows.Scan(&r.ID, &r.Category, &r.MatchID, &r.Body, &r.Page, &r.Username,
			&r.Player, &r.UserAgent, &r.Version, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// printBugReports writes the newest reports for a person to read, newest first.
func printBugReports(w io.Writer, store *Store, limit int) error {
	reports, err := store.BugReports(limit)
	if err != nil {
		return err
	}
	if len(reports) == 0 {
		fmt.Fprintln(w, "No bug reports.")
		return nil
	}
	for _, r := range reports {
		who := r.Username
		if who == "" {
			who = "guest"
		}
		fmt.Fprintf(w, "#%d  %s  %s  by %s", r.ID, r.CreatedAt.Local().Format("2006-01-02 15:04"), r.Category, who)
		if r.MatchID != "" {
			fmt.Fprintf(w, "  match %s", r.MatchID)
		}
		fmt.Fprintf(w, "\n    from %s, version %s\n    %s\n", orDash(r.Page), r.Version, r.UserAgent)
		for _, line := range strings.Split(r.Body, "\n") {
			fmt.Fprintf(w, "    > %s\n", line)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
