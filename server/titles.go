package main

import (
	"math"
	"net/http"
	"strings"
)

// Titles. Each rating mark in ratingUnlocks is a title, earned by the
// highest rating reached in either Eat the King or War Mode and kept for life.
// Names are in the web dictionary (titles.<id>); the short tag shown beside a
// name is there too. Must match TITLES in web/src/titles.ts.
var titles = []struct {
	at int
	id string
}{
	{1150, "club"},
	{1300, "expert"},
	{1450, "candidate"},
	{1600, "fog"},
	{1800, "im"},
	{2000, "gm"},
	{2500, "king"},
}

// titleFor is the title for a peak rating, or "" below the first.
func titleFor(peak int) string {
	out := ""
	for _, t := range titles {
		if peak >= t.at {
			out = t.id
		}
	}
	return out
}

// Titles returns the titles of the named players who have one.
func (s *Store) Titles(names []string) (map[string]string, error) {
	lower := make([]string, len(names))
	for i, n := range names {
		lower[i] = strings.ToLower(n)
	}
	rows, err := s.db.Query(`
		SELECT u.username, GREATEST(u.rating, u.four_rating),
			COALESCE(MAX(substring(x.reason FROM 8)::int), 0)
		FROM users u
		LEFT JOIN unlocks x ON x.user_id = u.id AND x.reason ~ '^rating:[0-9]+$'
		WHERE lower(u.username) = ANY($1)
		GROUP BY u.id`, lower)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name string
		var now float64
		var recorded int
		if err := rows.Scan(&name, &now, &recorded); err != nil {
			return nil, err
		}
		if t := titleFor(max(int(math.Round(now)), recorded)); t != "" {
			out[name] = t
		}
	}
	return out, rows.Err()
}

func titleRoutes(mux *http.ServeMux, store *Store) {
	// Anyone may look up titles, up to 100 names at once: ?u=ann,ben.
	mux.HandleFunc("GET /api/titles", func(w http.ResponseWriter, r *http.Request) {
		var names []string
		for _, n := range strings.Split(r.URL.Query().Get("u"), ",") {
			if n != "" && len(names) < 100 {
				names = append(names, n)
			}
		}
		out := map[string]string{}
		if len(names) > 0 {
			var err error
			if out, err = store.Titles(names); serverError(w, err) {
				return
			}
		}
		w.Header().Set("Cache-Control", "max-age=60")
		writeJSON(w, out)
	})
}
