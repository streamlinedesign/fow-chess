package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// setFamilies are the sets a player picks for White and for Black. Must match
// SET_IDS in web/src/sets.ts. "classic" is Eat the King's own: cream for White,
// ink for Black. A colour is drawn with light lines as White and dark lines as
// Black, and a theme in its light half as White and its dark half as Black, so
// the two sides never look alike even when both players pick the same set.
var setFamilies = []string{
	"classic",
	"red", "blue", "green", "violet",
	"galaxy", "blueprint", "holy", "ancient", "cyberpunk",
}

// pieceSets are the one-colour sets, each of which is also a row of avatars.
// Must match SETS in web/src/sets.ts. The plain colours ("red") are the
// four-player defaults, kept for avatars picked before the light and dark lines.
var pieceSets = []string{
	"cream", "ink",
	"red", "blue", "green", "violet",
	"red-light", "blue-light", "green-light", "violet-light",
	"red-ink", "blue-ink", "green-ink", "violet-ink",
	"galaxy-light", "galaxy-dark", "blueprint-light", "blueprint-dark",
	"holy-light", "holy-dark", "ancient-light", "ancient-dark",
	"cyberpunk-light", "cyberpunk-dark",
}

// tileSkins are the board skins for your half: "" is the plain board. Must match TILE_IDS in web/src/sets.ts.
var tileSkins = []string{"", "galaxy", "blueprint", "holy", "ancient", "cyberpunk"}

// avatarKinds are the pieces an avatar can be: the usual six, the jester pawn and the prince.
const avatarKinds = "KQRBNPJX"

// skin is how a player's own pieces and squares look, to everyone in their
// games: their set as White and as Black, and the tiles on their half (first three rows with four players).
type skin struct {
	White string `json:"white"`
	Black string `json:"black"`
	// The skins for the light and dark squares on your half (your first three rows with four players); "" for plain.
	LightTiles string `json:"lightTiles"`
	DarkTiles  string `json:"darkTiles"`
}

// plain is whether the skin draws exactly like the default.
func (s skin) plain() bool {
	return (s.White == "" || s.White == "classic") && (s.Black == "" || s.Black == "classic") &&
		s.LightTiles == "" && s.DarkTiles == ""
}

func (s skin) valid() bool {
	return slices.Contains(setFamilies, s.White) && slices.Contains(setFamilies, s.Black) &&
		slices.Contains(tileSkins, s.LightTiles) && slices.Contains(tileSkins, s.DarkTiles)
}

// clean drops what's no longer offered, so an old choice falls back to the default.
func (s *skin) clean() {
	if !slices.Contains(setFamilies, s.White) {
		s.White = "classic"
	}
	if !slices.Contains(setFamilies, s.Black) {
		s.Black = "classic"
	}
	if !slices.Contains(tileSkins, s.LightTiles) {
		s.LightTiles = ""
	}
	if !slices.Contains(tileSkins, s.DarkTiles) {
		s.DarkTiles = ""
	}
}

// cleanSkin is a skin a browser sent, or nil when it draws like the default or isn't allowed.
func cleanSkin(s *skin) *skin {
	if s == nil || !s.valid() || s.plain() {
		return nil
	}
	c := *s
	return &c
}

// prefs are a player's own choices for how the game looks and which side they play.
type prefs struct {
	skin
	Side string `json:"side"` // "w", "b", or "" for no preference
	// Plain turns every skin off on this player's screen: classic pieces and
	// the plain board for everyone but bots. The picks are kept.
	Plain bool `json:"plain"`
	// Primary was the set for your own pieces before White and Black sets.
	// Read once to carry an old choice over, never written.
	Primary string `json:"primary,omitempty"`
}

// familyOf is the set family a one-colour or old two-sided set belongs to.
func familyOf(id string) string {
	switch id {
	case "cream", "ink":
		return "classic"
	}
	for _, f := range setFamilies {
		if id == f || strings.HasPrefix(id, f+"-") {
			return f
		}
	}
	return "classic"
}

func (p *prefs) fill() {
	if p.Primary != "" && p.White == "" && p.Black == "" {
		p.White, p.Black = familyOf(p.Primary), familyOf(p.Primary)
	}
	p.Primary = ""
	if p.White == "" {
		p.White = "classic"
	}
	if p.Black == "" {
		p.Black = "classic"
	}
}

func (p prefs) valid() bool {
	return p.skin.valid() && (p.Side == "" || p.Side == "w" || p.Side == "b")
}

// validAvatar: "" for the default pawn, or "<set>/<piece>" from a one-colour set.
func validAvatar(a string) bool {
	if a == "" {
		return true
	}
	set, kind, ok := strings.Cut(a, "/")
	return ok && slices.Contains(pieceSets, set) && len(kind) == 1 && strings.Contains(avatarKinds, kind)
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
	l.fill()
	// A set that's since gone falls back to the default.
	l.skin.clean()
	if l.Side != "w" && l.Side != "b" {
		l.Side = ""
	}
	if !validAvatar(l.Avatar) {
		l.Avatar = ""
	}
	// What's still locked falls back to the default too.
	u, err := s.Unlocks(id)
	if err != nil {
		return l, err
	}
	u.strip(&l.skin)
	if !u.avatar(l.Avatar) {
		l.Avatar = ""
	}
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
		l.Primary = ""
		l.fill()
		if !l.prefs.valid() || !validAvatar(l.Avatar) {
			jsonError(w, "Those settings aren't allowed.", http.StatusBadRequest)
			return
		}
		u, err := s.store.Unlocks(me.ID)
		if serverError(w, err) {
			return
		}
		if !u.allows(l) {
			jsonError(w, "Beat the bot that wears that skin to unlock it.", http.StatusForbidden)
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
