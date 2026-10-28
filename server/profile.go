package main

import (
	"net/http"
	"strconv"
	"strings"

	"chess/engine"
)

// gameSummary is one game in a profile's lists.
type gameSummary struct {
	MatchID  string   `json:"matchId"`
	Round    int      `json:"round"`
	Color    string   `json:"color"`    // the profile owner's color
	Opponent string   `json:"opponent"` // username, or "" for a guest or an empty seat
	Waiting  bool     `json:"waiting"`  // no opponent has joined yet
	Settings Settings `json:"settings"`
	Plies    int      `json:"plies"`
	Turn     string   `json:"turn"`
	Result   string   `json:"result,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	Rated    bool     `json:"rated"`
	// RatingDiff is the profile owner's rating change, once a rated game is over.
	RatingDiff *int  `json:"ratingDiff,omitempty"`
	UpdatedAt  int64 `json:"updatedAt"` // unix milliseconds
	// Sniper marks a game won on the winner's first move.
	Sniper bool `json:"sniper,omitempty"`
}

type profileResponse struct {
	Username string        `json:"username"`
	Rating   int           `json:"rating"`
	Joined   int64         `json:"joined"` // unix milliseconds
	Online   bool          `json:"online"`
	LastSeen int64         `json:"lastSeen,omitempty"` // unix milliseconds; left out when hidden
	Playing  string        `json:"playing,omitempty"`  // the match they're playing right now
	Stats    record        `json:"stats"`
	Current  []gameSummary `json:"current"`
	Finished []gameSummary `json:"finished"`
	// For a signed-in visitor looking at someone else: are they friends, and
	// what can they do.
	Friend    string  `json:"friend,omitempty"`
	CanOffer  bool    `json:"canOffer,omitempty"`
	CanAdd    bool    `json:"canAdd,omitempty"`
	ScoreWith *record `json:"scoreWith,omitempty"` // the visitor's score against a friend
	// FormWith is the visitor's last results against a friend, newest first: W, L or D.
	FormWith []string `json:"formWith,omitempty"`
	Rank     int      `json:"rank,omitempty"` // place on the leaderboard, once rated
	// HideOnline is only sent to the player themselves.
	HideOnline bool `json:"hideOnline,omitempty"`
}

// summarize describes a game from the side of the player with userID.
func summarize(g gameRecord, userID int64) gameSummary {
	color := white
	if g.Users[black] == userID {
		color = black
	}
	other := 1 - color
	plies := len(strings.Fields(g.Moves))
	return gameSummary{
		MatchID: g.MatchID, Round: g.Round, Color: colorNames[color],
		Opponent: [2]string{g.whiteName, g.blackName}[other], Waiting: g.Seats[other] == "",
		Settings: g.Settings, Plies: plies, Turn: colorNames[plies%2],
		Result: g.Result, Reason: g.Reason, Rated: g.Rated, RatingDiff: g.RatingDiff[color],
		UpdatedAt: g.UpdatedAt.UnixMilli(), Sniper: g.sniper(),
	}
}

// visitor fills in what a signed-in visitor can do on someone else's profile.
func visitor(store *Store, me, them int64, resp *profileResponse) error {
	state, err := store.Friendship(me, them)
	if err != nil {
		return err
	}
	p, err := store.Privacy(them)
	if err != nil {
		return err
	}
	resp.Friend = state
	resp.CanOffer = state == friendsWith || p.OffersFrom == "anyone"
	resp.CanAdd = state != notFriends || p.RequestsFrom == "anyone"
	if state == friendsWith {
		score, err := store.Record(me, them)
		if err != nil {
			return err
		}
		resp.ScoreWith = &score
		games, err := store.GamesBetween(me, them, 5)
		if err != nil {
			return err
		}
		for _, g := range games {
			s := summarize(g, me)
			form := "D"
			if s.Result != "1/2-1/2" {
				form = "L"
				if (s.Result == "1-0") == (s.Color == "w") {
					form = "W"
				}
			}
			resp.FormWith = append(resp.FormWith, form)
		}
	}
	return nil
}

// replayFrame is the full board after each move of a finished game.
type replayFrame struct {
	Board    string            `json:"board"`
	Last     *lastMove         `json:"last"`
	Check    string            `json:"check,omitempty"`
	Captured map[string]string `json:"captured"`
}

type replayResponse struct {
	MatchID  string            `json:"matchId"`
	Round    int               `json:"round"`
	Settings Settings          `json:"settings"`
	Players  map[string]string `json:"players"` // usernames, "" for guests
	Result   string            `json:"result"`
	Reason   string            `json:"reason"`
	History  []string          `json:"history"`
	Frames   []replayFrame     `json:"frames"`
	Ended    int64             `json:"ended"` // unix milliseconds
	Sniper   bool              `json:"sniper,omitempty"`
}

const profileGames = 200

func profileRoutes(mux *http.ServeMux, store *Store, auth *Auth, live *Live) {
	mux.HandleFunc("GET /api/users/{name}", func(w http.ResponseWriter, r *http.Request) {
		u, err := store.UserByName(r.PathValue("name"))
		if err != nil {
			jsonError(w, "Couldn't load the profile.", http.StatusInternalServerError)
			return
		}
		if u == nil {
			jsonError(w, "No such player.", http.StatusNotFound)
			return
		}
		games, err := store.GamesOf(u.ID, profileGames)
		if err != nil {
			jsonError(w, "Couldn't load the profile.", http.StatusInternalServerError)
			return
		}
		p, err := store.Person(u.ID)
		if err != nil {
			jsonError(w, "Couldn't load the profile.", http.StatusInternalServerError)
			return
		}
		stats, err := store.Record(u.ID, 0)
		if err != nil {
			jsonError(w, "Couldn't load the profile.", http.StatusInternalServerError)
			return
		}
		card := live.card(*p)
		resp := profileResponse{
			Username: u.Username, Rating: int(u.Rating + 0.5), Joined: u.CreatedAt.UnixMilli(),
			Online: card.Online, LastSeen: card.LastSeen, Playing: card.Playing, Stats: stats,
			Current: []gameSummary{}, Finished: []gameSummary{},
		}
		if rank, err := store.Rank(u.ID); err == nil && rank != nil {
			resp.Rank = rank.Rank
		}
		me := auth.user(r)
		if me != nil && me.ID == u.ID {
			// You're here, so you're online, whatever others are shown.
			resp.Online, resp.HideOnline = true, p.HideOnline
		}
		if me != nil && me.ID != u.ID {
			if err := visitor(store, me.ID, u.ID, &resp); err != nil {
				jsonError(w, "Couldn't load the profile.", http.StatusInternalServerError)
				return
			}
		}
		for _, g := range games {
			s := summarize(g, u.ID)
			if g.Result == "" {
				resp.Current = append(resp.Current, s)
			} else {
				resp.Finished = append(resp.Finished, s)
			}
		}
		writeJSON(w, resp)
	})

	// The Top 100, plus where the signed-in player stands if they're further down.
	mux.HandleFunc("GET /api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		top, err := store.Leaderboard(100)
		if err != nil {
			jsonError(w, "Couldn't load the leaderboard.", http.StatusInternalServerError)
			return
		}
		resp := struct {
			Top []leader `json:"top"`
			Me  *leader  `json:"me"`
		}{Top: top}
		if u := auth.user(r); u != nil {
			if resp.Me, err = store.Rank(u.ID); err != nil {
				jsonError(w, "Couldn't load the leaderboard.", http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, resp)
	})

	// A finished game, move by move with the whole board visible. Games still
	// being played aren't served, since that would lift the fog.
	mux.HandleFunc("GET /api/games/{id}/{round}", func(w http.ResponseWriter, r *http.Request) {
		round, err := strconv.Atoi(r.PathValue("round"))
		if err != nil {
			jsonError(w, "No such game.", http.StatusNotFound)
			return
		}
		rec, err := store.Game(strings.ToUpper(r.PathValue("id")), round)
		if err != nil {
			jsonError(w, "Couldn't load the game.", http.StatusInternalServerError)
			return
		}
		if rec == nil {
			jsonError(w, "No such game.", http.StatusNotFound)
			return
		}
		if rec.Result == "" {
			jsonError(w, "This game is still being played.", http.StatusForbidden)
			return
		}
		resp, err := buildReplay(rec)
		if err != nil {
			jsonError(w, "Couldn't load the game.", http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	})
}

func buildReplay(rec *gameRecord) (*replayResponse, error) {
	g, err := engine.NewGameFromFEN(startFrom(rec.StartFEN), rec.Settings.rules())
	if err != nil {
		return nil, err
	}
	frame := func() replayFrame {
		v := viewFor(g, -1, false, false)
		return replayFrame{Board: v.Board, Last: v.Last, Check: v.Check, Captured: v.Captured}
	}
	resp := &replayResponse{
		MatchID: rec.MatchID, Round: rec.Round, Settings: rec.Settings,
		Players: map[string]string{"w": rec.whiteName, "b": rec.blackName},
		Result:  rec.Result, Reason: rec.Reason, History: []string{}, Frames: []replayFrame{frame()},
		Ended: rec.UpdatedAt.UnixMilli(), Sniper: rec.sniper(),
	}
	for _, uci := range strings.Fields(rec.Moves) {
		pm, err := g.PlayUCI(uci)
		if err != nil {
			return nil, err
		}
		resp.History = append(resp.History, pm.SAN)
		resp.Frames = append(resp.Frames, frame())
	}
	return resp, nil
}
