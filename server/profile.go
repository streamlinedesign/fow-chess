package main

import (
	"net/http"
	"strconv"
	"strings"

	"chess/engine"
)

// gameSummary is one game in a profile's lists.
type gameSummary struct {
	MatchID   string   `json:"matchId"`
	Round     int      `json:"round"`
	Color     string   `json:"color"`    // the profile owner's color
	Opponent  string   `json:"opponent"` // username, or "" for a guest or an empty seat
	Waiting   bool     `json:"waiting"`  // no opponent has joined yet
	Settings  Settings `json:"settings"`
	Plies     int      `json:"plies"`
	Turn      string   `json:"turn"`
	Result    string   `json:"result,omitempty"`
	Reason    string   `json:"reason,omitempty"`
	UpdatedAt int64    `json:"updatedAt"` // unix milliseconds
}

type profileResponse struct {
	Username string        `json:"username"`
	Rating   int           `json:"rating"`
	Joined   int64         `json:"joined"` // unix milliseconds
	Current  []gameSummary `json:"current"`
	Finished []gameSummary `json:"finished"`
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
}

const profileGames = 200

func profileRoutes(mux *http.ServeMux, store *Store, auth *Auth) {
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
		resp := profileResponse{
			Username: u.Username, Rating: int(u.Rating + 0.5), Joined: u.CreatedAt.UnixMilli(),
			Current: []gameSummary{}, Finished: []gameSummary{},
		}
		for _, g := range games {
			color := white
			if g.Users[black] == u.ID {
				color = black
			}
			other := 1 - color
			plies := len(strings.Fields(g.Moves))
			s := gameSummary{
				MatchID: g.MatchID, Round: g.Round, Color: colorNames[color],
				Opponent: [2]string{g.whiteName, g.blackName}[other], Waiting: g.Seats[other] == "",
				Settings: g.Settings, Plies: plies, Turn: colorNames[plies%2],
				Result: g.Result, Reason: g.Reason, UpdatedAt: g.UpdatedAt.UnixMilli(),
			}
			if g.Result == "" {
				resp.Current = append(resp.Current, s)
			} else {
				resp.Finished = append(resp.Finished, s)
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
	g := engine.NewGameWithRules(rec.Settings.rules())
	frame := func() replayFrame {
		v := viewFor(g, -1, false, false)
		return replayFrame{Board: v.Board, Last: v.Last, Check: v.Check, Captured: v.Captured}
	}
	resp := &replayResponse{
		MatchID: rec.MatchID, Round: rec.Round, Settings: rec.Settings,
		Players: map[string]string{"w": rec.whiteName, "b": rec.blackName},
		Result:  rec.Result, Reason: rec.Reason, History: []string{}, Frames: []replayFrame{frame()},
		Ended: rec.UpdatedAt.UnixMilli(),
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
