package main

import (
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Social serves friends, messages between friends, match offers and the bell.
// Only signed-in players have any of it.
type Social struct {
	store *Store
	live  *Live
	hub   *Hub
	auth  *Auth
}

// notify stores a notification for the bell and tells the player's open tabs.
func (s *Social) notify(n notification) {
	if err := s.store.AddNotification(n); err != nil {
		log.Printf("notification: %v", err)
		return
	}
	s.live.send(n.UserID, liveEvent{Type: "notification"})
}

// signedIn returns the signed-in user, or answers 401 and returns nil.
func (s *Social) signedIn(w http.ResponseWriter, r *http.Request) *User {
	u := s.auth.user(r)
	if u == nil {
		jsonError(w, "Log in first.", http.StatusUnauthorized)
	}
	return u
}

// other returns the player named in the path, or answers and returns nil.
func (s *Social) other(w http.ResponseWriter, r *http.Request, me *User) *User {
	o, err := s.store.UserByName(r.PathValue("name"))
	switch {
	case err != nil:
		jsonError(w, "Something went wrong.", http.StatusInternalServerError)
	case o == nil:
		jsonError(w, "No such player.", http.StatusNotFound)
	case o.ID == me.ID:
		jsonError(w, "That's you.", http.StatusBadRequest)
	default:
		return o
	}
	return nil
}

func serverError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	log.Printf("social: %v", err)
	jsonError(w, "Something went wrong.", http.StatusInternalServerError)
	return true
}

const (
	searchLimit       = 20
	messagePage       = 100
	notificationsPage = 50
	chatGames         = 50
)

func (s *Social) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/live", func(w http.ResponseWriter, r *http.Request) {
		if me := s.signedIn(w, r); me != nil {
			s.live.serve(w, r, me)
		}
	})

	mux.HandleFunc("GET /api/users/search", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		type result struct {
			userCard
			Friend string `json:"friend"`
		}
		out := []result{}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, out)
			return
		}
		ps, err := s.store.SearchUsers(q, searchLimit+1)
		if serverError(w, err) {
			return
		}
		for _, p := range ps {
			if p.ID == me.ID || len(out) == searchLimit {
				continue
			}
			state, err := s.store.Friendship(me.ID, p.ID)
			if serverError(w, err) {
				return
			}
			out = append(out, result{s.live.card(p), state})
		}
		writeJSON(w, out)
	})

	// Friends, online ones first, and the requests you've sent.
	mux.HandleFunc("GET /api/friends", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		friends, err := s.store.Friends(me.ID)
		if serverError(w, err) {
			return
		}
		sent, err := s.store.SentRequests(me.ID)
		if serverError(w, err) {
			return
		}
		cards := s.live.cards(friends)
		online := []userCard{}
		offline := []userCard{}
		for _, c := range cards {
			if c.Online {
				online = append(online, c)
			} else {
				offline = append(offline, c)
			}
		}
		writeJSON(w, map[string]any{"friends": append(online, offline...), "sent": s.live.cards(sent)})
	})

	// Sends a friend request, or accepts theirs.
	mux.HandleFunc("POST /api/friends/{name}", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		o := s.other(w, r, me)
		if o == nil {
			return
		}
		state, err := s.store.Friendship(me.ID, o.ID)
		if serverError(w, err) {
			return
		}
		if state == notFriends {
			p, err := s.store.Privacy(o.ID)
			if serverError(w, err) {
				return
			}
			if p.RequestsFrom == "nobody" {
				jsonNamedError(w, o.Username+" isn't taking friend requests.", "not_taking_requests", o.Username, http.StatusForbidden)
				return
			}
		}
		state, changed, err := s.store.RequestFriend(me.ID, o.ID)
		if serverError(w, err) {
			return
		}
		if changed {
			kind := notifyFriendRequest
			if state == friendsWith {
				kind = notifyFriendAccepted
				s.live.send(me.ID, liveEvent{Type: "friends"})
			}
			go s.notify(notification{UserID: o.ID, Kind: kind, ActorID: me.ID})
			s.live.send(o.ID, liveEvent{Type: "friends"})
		}
		writeJSON(w, map[string]string{"friend": state})
	})

	// Removes a friend, cancels your request, or declines theirs.
	mux.HandleFunc("DELETE /api/friends/{name}", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		o := s.other(w, r, me)
		if o == nil {
			return
		}
		if serverError(w, s.store.Unfriend(me.ID, o.ID)) {
			return
		}
		s.live.send(me.ID, liveEvent{Type: "friends"})
		s.live.send(o.ID, liveEvent{Type: "friends"})
		writeJSON(w, map[string]string{"friend": notFriends})
	})

	mux.HandleFunc("GET /api/messages", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		convs, err := s.store.Conversations(me.ID)
		if serverError(w, err) {
			return
		}
		type item struct {
			With   userCard `json:"with"`
			Last   message  `json:"last"`
			Unread int      `json:"unread"`
		}
		out := make([]item, len(convs))
		for i, c := range convs {
			out[i] = item{s.live.card(c.With), c.Last, c.Unread}
		}
		writeJSON(w, out)
	})

	// One chat: messages (older pages with ?before=<id>), your games against
	// each other, and the score between you.
	mux.HandleFunc("GET /api/messages/{name}", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		o := s.other(w, r, me)
		if o == nil {
			return
		}
		before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		msgs, err := s.store.Messages(me.ID, o.ID, before, messagePage)
		if serverError(w, err) {
			return
		}
		resp := map[string]any{"messages": msgs, "more": len(msgs) == messagePage}
		if before == 0 {
			p, err := s.store.Person(o.ID)
			if serverError(w, err) {
				return
			}
			state, err := s.store.Friendship(me.ID, o.ID)
			if serverError(w, err) {
				return
			}
			score, err := s.store.Record(me.ID, o.ID)
			if serverError(w, err) {
				return
			}
			games, err := s.store.GamesBetween(me.ID, o.ID, chatGames)
			if serverError(w, err) {
				return
			}
			mine, theirs, err := s.store.ChatDeletion(me.ID, o.ID)
			if serverError(w, err) {
				return
			}
			summaries := make([]gameSummary, len(games))
			for i, g := range games {
				summaries[i] = summarize(g, me.ID)
			}
			resp["with"] = s.live.card(*p)
			resp["friend"] = state
			resp["score"] = score
			resp["games"] = summaries
			resp["deletion"] = map[string]bool{"mine": mine, "theirs": theirs}
		}
		writeJSON(w, resp)
	})

	mux.HandleFunc("POST /api/messages/{name}", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		o := s.other(w, r, me)
		if o == nil {
			return
		}
		text := strings.TrimSpace(body.Text)
		if text == "" || !utf8.ValidString(text) {
			jsonError(w, "Write something first.", http.StatusBadRequest)
			return
		}
		if utf8.RuneCountInString(text) > maxMessage {
			jsonError(w, "Messages are up to 1000 characters.", http.StatusBadRequest)
			return
		}
		state, err := s.store.Friendship(me.ID, o.ID)
		if serverError(w, err) {
			return
		}
		if state != friendsWith {
			jsonError(w, "Only friends can message each other.", http.StatusForbidden)
			return
		}
		id, err := s.store.SendMessage(me.ID, o.ID, text, 0)
		if serverError(w, err) {
			return
		}
		s.live.send(o.ID, liveEvent{Type: "message", User: me.Username})
		s.live.send(me.ID, liveEvent{Type: "message", User: o.Username})
		writeJSON(w, map[string]int64{"id": id})
	})

	mux.HandleFunc("POST /api/messages/{name}/read", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var body struct {
			UpTo int64 `json:"upTo"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		o := s.other(w, r, me)
		if o == nil || serverError(w, s.store.MarkRead(me.ID, o.ID, body.UpTo)) {
			return
		}
		// The unread count in your other tabs goes down too.
		s.live.send(me.ID, liveEvent{Type: "read"})
		w.WriteHeader(http.StatusNoContent)
	})

	// Asks to delete a chat, or takes the request back. Once both ask, it's gone.
	mux.HandleFunc("POST /api/messages/{name}/delete", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var body struct {
			Want bool `json:"want"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		o := s.other(w, r, me)
		if o == nil {
			return
		}
		deleted, err := s.store.AskToDeleteChat(me.ID, o.ID, body.Want)
		if serverError(w, err) {
			return
		}
		s.live.send(o.ID, liveEvent{Type: "message", User: me.Username})
		s.live.send(me.ID, liveEvent{Type: "message", User: o.Username})
		writeJSON(w, map[string]bool{"deleted": deleted})
	})

	mux.HandleFunc("POST /api/offers", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var body struct {
			To       string   `json:"to"`
			Settings Settings `json:"settings"`
			Color    string   `json:"color"` // yours: w, b or random
		}
		if !readJSON(w, r, &body) {
			return
		}
		if body.Color == "" {
			body.Color = "random"
		}
		if body.Color != "w" && body.Color != "b" && body.Color != "random" {
			jsonError(w, "Pick white, black or random.", http.StatusBadRequest)
			return
		}
		body.Settings.normalize()
		if !body.Settings.valid() {
			jsonError(w, "Those match settings aren't available.", http.StatusBadRequest)
			return
		}
		o, err := s.store.UserByName(body.To)
		if serverError(w, err) {
			return
		}
		if o == nil || o.ID == me.ID {
			jsonError(w, "No such player.", http.StatusNotFound)
			return
		}
		state, err := s.store.Friendship(me.ID, o.ID)
		if serverError(w, err) {
			return
		}
		p, err := s.store.Privacy(o.ID)
		if serverError(w, err) {
			return
		}
		if p.OffersFrom == "friends" && state != friendsWith {
			jsonNamedError(w, o.Username+" only takes match offers from friends.", "offers_from_friends", o.Username, http.StatusForbidden)
			return
		}
		id, err := s.store.CreateOffer(me.ID, o.ID, body.Settings, body.Color)
		if serverError(w, err) {
			return
		}
		// Between friends the offer also shows up in their chat.
		if state == friendsWith {
			if _, err := s.store.SendMessage(me.ID, o.ID, "", id); serverError(w, err) {
				return
			}
			s.live.send(o.ID, liveEvent{Type: "message", User: me.Username})
			s.live.send(me.ID, liveEvent{Type: "message", User: o.Username})
		}
		go s.notify(notification{UserID: o.ID, Kind: notifyOffer, ActorID: me.ID, OfferID: id})
		writeJSON(w, map[string]int64{"id": id})
	})

	// Accept starts the match, with the colors the offer asked for.
	mux.HandleFunc("POST /api/offers/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		o, err := s.store.Offer(id)
		if serverError(w, err) {
			return
		}
		action := r.PathValue("action")
		mine := o != nil && ((action == "cancel" && o.From == me.ID) || ((action == "accept" || action == "decline") && o.To == me.ID))
		if !mine {
			jsonError(w, "No such offer.", http.StatusNotFound)
			return
		}
		status := map[string]string{"accept": "accepted", "decline": "declined", "cancel": "cancelled"}[action]
		ok, err := s.store.CloseOffer(id, status, "")
		if serverError(w, err) {
			return
		}
		if !ok {
			jsonError(w, "This offer has already been answered.", http.StatusConflict)
			return
		}
		other := o.To
		if other == me.ID {
			other = o.From
		}
		resp := map[string]string{"status": status}
		switch action {
		case "accept":
			from, err := s.store.UserByID(o.From)
			if serverError(w, err) {
				return
			}
			if from == nil {
				jsonError(w, "That player has left.", http.StatusGone)
				return
			}
			pw, pb := userPlayer(from, ""), userPlayer(me, "")
			if o.Color == "b" || (o.Color == "random" && rand.IntN(2) == 1) {
				pw, pb = pb, pw
			}
			m := s.hub.CreateBetween(pw, pb, o.Settings)
			if serverError(w, s.store.SetOfferMatch(id, m.id)) {
				return
			}
			resp["matchId"] = m.id
			go s.notify(notification{UserID: o.From, Kind: notifyOfferAccepted, ActorID: me.ID, OfferID: id, MatchID: m.id, Round: 1})
		case "decline":
			go s.notify(notification{UserID: o.From, Kind: notifyOfferDeclined, ActorID: me.ID, OfferID: id})
		}
		s.live.send(me.ID, liveEvent{Type: "offer", MatchID: resp["matchId"]})
		s.live.send(other, liveEvent{Type: "offer", MatchID: resp["matchId"], User: me.Username})
		writeJSON(w, resp)
	})

	mux.HandleFunc("GET /api/notifications", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		items, err := s.store.Notifications(me.ID, notificationsPage)
		if serverError(w, err) {
			return
		}
		unread, err := s.store.UnreadNotifications(me.ID)
		if serverError(w, err) {
			return
		}
		chats, err := s.store.UnreadMessages(me.ID)
		if serverError(w, err) {
			return
		}
		writeJSON(w, map[string]any{"items": items, "unread": unread, "unreadChats": chats})
	})

	mux.HandleFunc("POST /api/notifications/read", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil || serverError(w, s.store.MarkNotificationsRead(me.ID)) {
			return
		}
		s.live.send(me.ID, liveEvent{Type: "read"})
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		p, err := s.store.Privacy(me.ID)
		if serverError(w, err) {
			return
		}
		writeJSON(w, p)
	})

	mux.HandleFunc("PUT /api/settings", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var p privacy
		if !readJSON(w, r, &p) {
			return
		}
		if !p.valid() {
			jsonError(w, "Those settings aren't allowed.", http.StatusBadRequest)
			return
		}
		if serverError(w, s.store.SetPrivacy(me.ID, p)) {
			return
		}
		writeJSON(w, p)
	})
}
