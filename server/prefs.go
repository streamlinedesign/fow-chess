package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// The piece sets a player can pick. Must match SETS in web/src/sets.ts.
// "classic" is Eat the King's own: cream for White, ink for Black. The rest
// are one colour each, and every piece of those is also an avatar.
var pieceSets = []string{
	"classic",
	"cream", "ink",
	"red", "blue", "green", "violet",
	"red-ink", "blue-ink", "green-ink", "violet-ink",
	// The themed sets, light for White and dark for Black like classic. Each
	// half is a one-colour set of its own, for avatars.
	"galaxy", "blueprint", "holy", "ancient",
	"galaxy-light", "galaxy-dark", "blueprint-light", "blueprint-dark",
	"holy-light", "holy-dark", "ancient-light", "ancient-dark",
}

// twoSided are the sets drawn differently for White and Black: never avatars.
var twoSided = []string{"classic", "galaxy", "blueprint", "holy", "ancient"}

// tileSkins are the board skins for your half: "" is the plain board. Must match TILE_IDS in web/src/sets.ts.
var tileSkins = []string{"", "galaxy", "blueprint", "holy", "ancient"}

// avatarKinds are the pieces an avatar can be: the usual six, the jester pawn and the prince.
const avatarKinds = "KQRBNPJX"

// prefs are a player's own choices for how the game looks and which side they play.
type prefs struct {
	Primary   string `json:"primary"`   // the set for your pieces
	Secondary string `json:"secondary"` // the set for your opponent's
	Side      string `json:"side"`      // "w", "b", or "" for no preference
	// The skins for the light and dark squares on your half; "" for plain.
	LightTiles string `json:"lightTiles"`
	DarkTiles  string `json:"darkTiles"`
	// Plain turns every skin off: classic pieces and the plain board. The picks are kept.
	Plain bool `json:"plain"`
}

func (p *prefs) fill() {
	if p.Primary == "" {
		p.Primary = "classic"
	}
	if p.Secondary == "" {
		p.Secondary = "classic"
	}
}

func (p prefs) valid() bool {
	return slices.Contains(pieceSets, p.Primary) && slices.Contains(pieceSets, p.Secondary) &&
		(p.Side == "" || p.Side == "w" || p.Side == "b") &&
		slices.Contains(tileSkins, p.LightTiles) && slices.Contains(tileSkins, p.DarkTiles)
}

// validAvatar: "" for the default pawn, or "<set>/<piece>" from a one-colour set.
func validAvatar(a string) bool {
	if a == "" {
		return true
	}
	set, kind, ok := strings.Cut(a, "/")
	return ok && !slices.Contains(twoSided, set) && slices.Contains(pieceSets, set) && len(kind) == 1 && strings.Contains(avatarKinds, kind)
}

type lookResponse struct {
	Avatar string `json:"avatar"`
	prefs
}

func (s *Store) Look(id int64) (lookResponse, error) {
	var l lookResponse
	var raw []byte
	err := s.db.QueryRow(`SELECT avatar, prefs FROM users WHERE id = $1`, id).Scan(&l.Avatar, &raw)
	if err != nil {
		return l, err
	}
	json.Unmarshal(raw, &l.prefs)
	// A set that's since gone falls back to the default.
	if !slices.Contains(pieceSets, l.Primary) {
		l.Primary = ""
	}
	if !slices.Contains(pieceSets, l.Secondary) {
		l.Secondary = ""
	}
	if l.Side != "w" && l.Side != "b" {
		l.Side = ""
	}
	if !slices.Contains(tileSkins, l.LightTiles) {
		l.LightTiles = ""
	}
	if !slices.Contains(tileSkins, l.DarkTiles) {
		l.DarkTiles = ""
	}
	if !validAvatar(l.Avatar) {
		l.Avatar = ""
	}
	l.fill()
	return l, nil
}

func (s *Store) SetLook(id int64, l lookResponse) error {
	raw, _ := json.Marshal(l.prefs)
	_, err := s.db.Exec(`UPDATE users SET avatar = $2, prefs = $3 WHERE id = $1`, id, l.Avatar, raw)
	return err
}

// Avatars returns the avatars of the named players who have picked one.
func (s *Store) Avatars(names []string) (map[string]string, error) {
	lower := make([]string, len(names))
	for i, n := range names {
		lower[i] = strings.ToLower(n)
	}
	rows, err := s.db.Query(`SELECT username, avatar FROM users WHERE lower(username) = ANY($1) AND avatar <> ''`, lower)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, avatar string
		if err := rows.Scan(&name, &avatar); err != nil {
			return nil, err
		}
		if validAvatar(avatar) {
			out[name] = avatar
		}
	}
	return out, rows.Err()
}

func (s *Social) prefRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/prefs", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var l lookResponse
		if !readJSON(w, r, &l) {
			return
		}
		l.fill()
		if !l.prefs.valid() || !validAvatar(l.Avatar) {
			jsonError(w, "Those settings aren't allowed.", http.StatusBadRequest)
			return
		}
		if serverError(w, s.store.SetLook(me.ID, l)) {
			return
		}
		writeJSON(w, l)
	})

	// Anyone may look up avatars, up to 100 names at once: ?u=ann,ben.
	mux.HandleFunc("GET /api/avatars", func(w http.ResponseWriter, r *http.Request) {
		var names []string
		for _, n := range strings.Split(r.URL.Query().Get("u"), ",") {
			if n != "" && len(names) < 100 {
				names = append(names, n)
			}
		}
		out := map[string]string{}
		if len(names) > 0 {
			var err error
			if out, err = s.store.Avatars(names); serverError(w, err) {
				return
			}
		}
		w.Header().Set("Cache-Control", "max-age=60")
		writeJSON(w, out)
	})
}
