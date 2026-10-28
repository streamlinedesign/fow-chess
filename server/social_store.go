package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

// person is another player as friends, search and messages show them.
type person struct {
	ID         int64
	Username   string
	Rating     float64
	LastSeen   time.Time
	HideOnline bool
}

// pair orders two user IDs the way friends and messages store them.
func pair(x, y int64) (int64, int64) {
	if x < y {
		return x, y
	}
	return y, x
}

func (s *Store) people(query string, args ...any) ([]person, error) {
	rows, err := s.db.Query(`SELECT u.id, u.username, u.rating, u.last_seen, u.hide_online FROM users u `+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []person{}
	for rows.Next() {
		var p person
		var seen sql.NullTime
		if err := rows.Scan(&p.ID, &p.Username, &p.Rating, &seen, &p.HideOnline); err != nil {
			return nil, err
		}
		p.LastSeen = seen.Time
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Person(id int64) (*person, error) {
	ps, err := s.people(`WHERE u.id = $1`, id)
	if err != nil || len(ps) == 0 {
		return nil, err
	}
	return &ps[0], nil
}

// SearchUsers finds players whose names start with q, shortest names first.
func (s *Store) SearchUsers(q string, limit int) ([]person, error) {
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(q))
	return s.people(`WHERE lower(u.username) LIKE $1 || '%' ORDER BY length(u.username), lower(u.username) LIMIT $2`, q, limit)
}

// Friendship states, as the profile buttons show them.
const (
	notFriends      = "none"
	friendsWith     = "friends"
	requestSent     = "sent"
	requestReceived = "received"
)

func (s *Store) Friendship(me, other int64) (string, error) {
	a, b := pair(me, other)
	var state string
	err := s.db.QueryRow(`SELECT CASE
		WHEN EXISTS (SELECT 1 FROM friends WHERE a = $1 AND b = $2) THEN 'friends'
		WHEN EXISTS (SELECT 1 FROM friend_requests WHERE from_id = $3 AND to_id = $4) THEN 'sent'
		WHEN EXISTS (SELECT 1 FROM friend_requests WHERE from_id = $4 AND to_id = $3) THEN 'received'
		ELSE 'none' END`, a, b, me, other).Scan(&state)
	return state, err
}

func (s *Store) Friends(me int64) ([]person, error) {
	return s.people(`JOIN friends f ON (f.a = $1 AND f.b = u.id) OR (f.b = $1 AND f.a = u.id) ORDER BY lower(u.username)`, me)
}

func (s *Store) FriendIDs(me int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT CASE WHEN a = $1 THEN b ELSE a END FROM friends WHERE a = $1 OR b = $1`, me)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SentRequests lists the people me asked to be friends, newest first.
func (s *Store) SentRequests(me int64) ([]person, error) {
	return s.people(`JOIN friend_requests r ON r.to_id = u.id WHERE r.from_id = $1 ORDER BY r.created_at DESC`, me)
}

// RequestFriend asks other to be friends, or accepts if they already asked.
// It returns the new state and whether anything changed.
func (s *Store) RequestFriend(me, other int64) (state string, changed bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	a, b := pair(me, other)
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM friends WHERE a = $1 AND b = $2)`, a, b).Scan(&exists); err != nil || exists {
		return friendsWith, false, err
	}
	res, err := tx.Exec(`DELETE FROM friend_requests WHERE from_id = $1 AND to_id = $2`, other, me)
	if err != nil {
		return "", false, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if _, err := tx.Exec(`INSERT INTO friends (a, b, created_at) VALUES ($1, $2, now())`, a, b); err != nil {
			return "", false, err
		}
		return friendsWith, true, tx.Commit()
	}
	res, err = tx.Exec(`INSERT INTO friend_requests (from_id, to_id, created_at) VALUES ($1, $2, now()) ON CONFLICT DO NOTHING`, me, other)
	if err != nil {
		return "", false, err
	}
	n, _ := res.RowsAffected()
	return requestSent, n > 0, tx.Commit()
}

// Unfriend removes a friendship or a request either way. The chat stays.
func (s *Store) Unfriend(me, other int64) error {
	a, b := pair(me, other)
	_, err := s.db.Exec(`WITH f AS (DELETE FROM friends WHERE a = $1 AND b = $2)
		DELETE FROM friend_requests WHERE (from_id = $3 AND to_id = $4) OR (from_id = $4 AND to_id = $3)`, a, b, me, other)
	return err
}

// privacy is what a player lets others see and do.
type privacy struct {
	HideOnline   bool   `json:"hideOnline"`
	OffersFrom   string `json:"offersFrom"`   // anyone or friends
	RequestsFrom string `json:"requestsFrom"` // anyone or nobody
}

func (p privacy) valid() bool {
	return (p.OffersFrom == "anyone" || p.OffersFrom == "friends") && (p.RequestsFrom == "anyone" || p.RequestsFrom == "nobody")
}

func (s *Store) Privacy(id int64) (privacy, error) {
	var p privacy
	err := s.db.QueryRow(`SELECT hide_online, offers_from, requests_from FROM users WHERE id = $1`, id).Scan(&p.HideOnline, &p.OffersFrom, &p.RequestsFrom)
	return p, err
}

func (s *Store) SetPrivacy(id int64, p privacy) error {
	_, err := s.db.Exec(`UPDATE users SET hide_online = $2, offers_from = $3, requests_from = $4 WHERE id = $1`, id, p.HideOnline, p.OffersFrom, p.RequestsFrom)
	return err
}

func (s *Store) TouchLastSeen(id int64) error {
	_, err := s.db.Exec(`UPDATE users SET last_seen = now() WHERE id = $1`, id)
	return err
}

func (s *Store) SetPassword(id int64, hash []byte) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = $2 WHERE id = $1`, id, hash)
	return err
}

func (s *Store) DeleteSessions(userID int64) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

// DeleteUser removes an account and everything that's only theirs. Their games
// stay, played by a guest, and so do the messages they sent until the other
// side's chat is gone too.
func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}

// record is a player's results, overall or against one opponent.
type record struct {
	Wins   int `json:"wins"`
	Draws  int `json:"draws"`
	Losses int `json:"losses"`
	// Wins and losses by capturing the king, in Eat the King games.
	AteKing    int `json:"ateKing"`
	KingGotAte int `json:"kingGotAte"`
}

// Record counts userID's finished games, or only those against `against` if it isn't 0.
func (s *Store) Record(userID, against int64) (record, error) {
	var r record
	err := s.db.QueryRow(`SELECT
			count(*) FILTER (WHERE (white_user = $1 AND result = '1-0') OR (black_user = $1 AND result = '0-1')),
			count(*) FILTER (WHERE result = '1/2-1/2'),
			count(*) FILTER (WHERE (white_user = $1 AND result = '0-1') OR (black_user = $1 AND result = '1-0')),
			count(*) FILTER (WHERE reason = 'capturing the king' AND ((white_user = $1 AND result = '1-0') OR (black_user = $1 AND result = '0-1'))),
			count(*) FILTER (WHERE reason = 'capturing the king' AND ((white_user = $1 AND result = '0-1') OR (black_user = $1 AND result = '1-0')))
		FROM games
		WHERE result <> '' AND (white_user = $1 OR black_user = $1)
			AND coalesce(white_user, 0) <> coalesce(black_user, 0)
			AND ($2 = 0 OR white_user = $2 OR black_user = $2)`, userID, against).Scan(&r.Wins, &r.Draws, &r.Losses, &r.AteKing, &r.KingGotAte)
	return r, err
}

// GamesBetween lists two players' finished games against each other, newest first.
func (s *Store) GamesBetween(x, y int64, limit int) ([]gameRecord, error) {
	return s.queryGames(`WHERE g.result <> '' AND ((g.white_user = $1 AND g.black_user = $2) OR (g.white_user = $2 AND g.black_user = $1))
		ORDER BY g.updated_at DESC LIMIT $3`, x, y, limit)
}

// offer is a match offer as stored.
type offer struct {
	ID        int64
	From, To  int64
	Settings  Settings
	Color     string // the offering player's: w, b or random
	Status    string
	MatchID   string
	CreatedAt time.Time
}

// offerView is an offer as the web app shows it.
type offerView struct {
	ID       int64    `json:"id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Settings Settings `json:"settings"`
	Color    string   `json:"color"` // the offering player's: w, b or random
	Status   string   `json:"status"`
	MatchID  string   `json:"matchId,omitempty"`
	At       int64    `json:"at"`
}

// CreateOffer offers a match, replacing any open offer from the same player to the same player.
func (s *Store) CreateOffer(from, to int64, settings Settings, color string) (int64, error) {
	js, err := jsonString(settings)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRow(`WITH old AS (UPDATE offers SET status = 'cancelled' WHERE from_id = $1 AND to_id = $2 AND status = 'open')
		INSERT INTO offers (from_id, to_id, settings, color, created_at) VALUES ($1, $2, $3, $4, now()) RETURNING id`, from, to, js, color).Scan(&id)
	return id, err
}

func (s *Store) Offer(id int64) (*offer, error) {
	var o offer
	var settings string
	var matchID sql.NullString
	err := s.db.QueryRow(`SELECT id, from_id, to_id, settings::text, color, status, match_id, created_at FROM offers WHERE id = $1`, id).
		Scan(&o.ID, &o.From, &o.To, &settings, &o.Color, &o.Status, &matchID, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parseJSON(settings, &o.Settings)
	o.MatchID = matchID.String
	return &o, nil
}

// CloseOffer moves an open offer to status. It returns false if the offer was no longer open.
func (s *Store) CloseOffer(id int64, status, matchID string) (bool, error) {
	res, err := s.db.Exec(`UPDATE offers SET status = $2, match_id = nullif($3, '') WHERE id = $1 AND status = 'open'`, id, status, matchID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// offerColumns and scanOffer read an offer joined with its players' names,
// from a LEFT JOIN on offers o, users ofrom, users oto.
const offerColumns = `o.id, o.settings::text, o.color, o.status, coalesce(o.match_id, ''), o.created_at, ofrom.username, oto.username`

func scanOffer(settings sql.NullString, v *offerView, at sql.NullTime) *offerView {
	if !settings.Valid {
		return nil
	}
	parseJSON(settings.String, &v.Settings)
	v.At = at.Time.UnixMilli()
	return v
}

func (s *Store) SetOfferMatch(id int64, matchID string) error {
	_, err := s.db.Exec(`UPDATE offers SET match_id = $2 WHERE id = $1`, id, matchID)
	return err
}

const maxMessage = 1000 // characters

// message is one line of a chat between friends: text, or a match offer.
type message struct {
	ID    int64      `json:"id"`
	From  string     `json:"from"`
	Text  string     `json:"text,omitempty"`
	Offer *offerView `json:"offer,omitempty"`
	At    int64      `json:"at"`
}

func (s *Store) SendMessage(from, to int64, text string, offerID int64) (int64, error) {
	a, b := pair(from, to)
	var id int64
	err := s.db.QueryRow(`INSERT INTO messages (a, b, sender, body, offer_id, created_at) VALUES ($1, $2, $3, $4, $5, now()) RETURNING id`,
		a, b, from, text, nullID(offerID)).Scan(&id)
	if err != nil {
		return 0, err
	}
	// Your own message counts as read, and writing again takes back a request to delete the chat.
	_, err = s.db.Exec(`INSERT INTO chat_state (user_id, other_id, last_read) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, other_id) DO UPDATE SET last_read = greatest(chat_state.last_read, $3), delete_requested = false`, from, to, id)
	return id, err
}

// Messages returns up to limit messages between two players before the message
// ID before (0 for the newest), oldest first.
func (s *Store) Messages(me, other, before int64, limit int) ([]message, error) {
	a, b := pair(me, other)
	rows, err := s.db.Query(`SELECT m.id, su.username, m.body, m.created_at, `+offerColumns+`
		FROM messages m
		JOIN users su ON su.id = m.sender
		LEFT JOIN offers o ON o.id = m.offer_id
		LEFT JOIN users ofrom ON ofrom.id = o.from_id
		LEFT JOIN users oto ON oto.id = o.to_id
		WHERE m.a = $1 AND m.b = $2 AND ($3 = 0 OR m.id < $3)
		ORDER BY m.id DESC LIMIT $4`, a, b, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []message{}
	for rows.Next() {
		var m message
		var at time.Time
		var ov offerView
		var oid sql.NullInt64
		var settings, color, status, matchID, ofrom, oto sql.NullString
		var oat sql.NullTime
		if err := rows.Scan(&m.ID, &m.From, &m.Text, &at, &oid, &settings, &color, &status, &matchID, &oat, &ofrom, &oto); err != nil {
			return nil, err
		}
		m.At = at.UnixMilli()
		ov.Color = color.String
		ov.ID, ov.Status, ov.MatchID, ov.From, ov.To = oid.Int64, status.String, matchID.String, ofrom.String, oto.String
		m.Offer = scanOffer(settings, &ov, oat)
		out = append(out, m)
	}
	slices.Reverse(out)
	return out, rows.Err()
}

// conversation is one chat in the Messages list.
type conversation struct {
	With   person
	Last   message
	Unread int
}

// Conversations lists me's chats, the one with the newest message first.
func (s *Store) Conversations(me int64) ([]conversation, error) {
	rows, err := s.db.Query(`
		WITH last AS (
			SELECT DISTINCT ON (a, b) id, a, b, sender, body, offer_id, created_at
			FROM messages WHERE a = $1 OR b = $1 ORDER BY a, b, id DESC
		)
		SELECT u.id, u.username, u.rating, u.last_seen, u.hide_online,
			l.id, su.username, l.body, l.offer_id IS NOT NULL, l.created_at,
			(SELECT count(*) FROM messages m WHERE m.a = l.a AND m.b = l.b AND m.sender <> $1
				AND m.id > coalesce((SELECT last_read FROM chat_state WHERE user_id = $1 AND other_id = u.id), 0))
		FROM last l
		JOIN users u ON u.id = CASE WHEN l.a = $1 THEN l.b ELSE l.a END
		JOIN users su ON su.id = l.sender
		ORDER BY l.id DESC`, me)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []conversation{}
	for rows.Next() {
		var c conversation
		var seen sql.NullTime
		var at time.Time
		var isOffer bool
		if err := rows.Scan(&c.With.ID, &c.With.Username, &c.With.Rating, &seen, &c.With.HideOnline,
			&c.Last.ID, &c.Last.From, &c.Last.Text, &isOffer, &at, &c.Unread); err != nil {
			return nil, err
		}
		c.With.LastSeen = seen.Time
		c.Last.At = at.UnixMilli()
		if isOffer {
			c.Last.Offer = &offerView{}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) MarkRead(me, other, upTo int64) error {
	_, err := s.db.Exec(`INSERT INTO chat_state (user_id, other_id, last_read) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, other_id) DO UPDATE SET last_read = greatest(chat_state.last_read, $3)`, me, other, upTo)
	return err
}

// UnreadMessages counts the chats with messages me hasn't read.
func (s *Store) UnreadMessages(me int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT count(DISTINCT m.sender) FROM messages m
		LEFT JOIN chat_state c ON c.user_id = $1 AND c.other_id = m.sender
		WHERE (m.a = $1 OR m.b = $1) AND m.sender <> $1 AND m.id > coalesce(c.last_read, 0)`, me).Scan(&n)
	return n, err
}

// ChatDeletion reports who has asked to delete the chat between me and other.
func (s *Store) ChatDeletion(me, other int64) (mine, theirs bool, err error) {
	err = s.db.QueryRow(`SELECT
		coalesce((SELECT delete_requested FROM chat_state WHERE user_id = $1 AND other_id = $2), false),
		coalesce((SELECT delete_requested FROM chat_state WHERE user_id = $2 AND other_id = $1), false)`, me, other).Scan(&mine, &theirs)
	return
}

// AskToDeleteChat records that me wants the chat with other deleted (or no
// longer does). Once both want it, the chat is deleted and deleted is true.
func (s *Store) AskToDeleteChat(me, other int64, want bool) (deleted bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	a, b := pair(me, other)
	// Lock the pair so two people agreeing at once both see each other's request.
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock($1, $2)`, int32(a), int32(b)); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`INSERT INTO chat_state (user_id, other_id, delete_requested) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, other_id) DO UPDATE SET delete_requested = $3`, me, other, want); err != nil {
		return false, err
	}
	var theirs bool
	if err := tx.QueryRow(`SELECT coalesce((SELECT delete_requested FROM chat_state WHERE user_id = $1 AND other_id = $2), false)`, other, me).Scan(&theirs); err != nil {
		return false, err
	}
	if want && theirs {
		if _, err := tx.Exec(`DELETE FROM messages WHERE a = $1 AND b = $2`, a, b); err != nil {
			return false, err
		}
		if _, err := tx.Exec(`DELETE FROM chat_state WHERE (user_id = $1 AND other_id = $2) OR (user_id = $2 AND other_id = $1)`, a, b); err != nil {
			return false, err
		}
		deleted = true
	}
	return deleted, tx.Commit()
}

// Notification kinds.
const (
	notifyFriendRequest  = "friend_request"
	notifyFriendAccepted = "friend_accepted"
	notifyOffer          = "offer"
	notifyOfferAccepted  = "offer_accepted"
	notifyOfferDeclined  = "offer_declined"
	notifyGameOver       = "game_over"
	notifyMove           = "move"
)

// notification is something for the bell.
type notification struct {
	UserID  int64
	Kind    string
	ActorID int64
	OfferID int64
	MatchID string
	Round   int
	Data    map[string]any
}

// notificationView is a notification as the bell shows it.
type notificationView struct {
	ID      int64           `json:"id"`
	Kind    string          `json:"kind"`
	Actor   string          `json:"actor,omitempty"`
	Offer   *offerView      `json:"offer,omitempty"`
	MatchID string          `json:"matchId,omitempty"`
	Round   int             `json:"round,omitempty"`
	Data    json.RawMessage `json:"data"`
	Read    bool            `json:"read"`
	At      int64           `json:"at"`
	// Pending is whether a friend request is still waiting for an answer.
	Pending bool `json:"pending,omitempty"`
}

func (s *Store) AddNotification(n notification) error {
	if n.Data == nil {
		n.Data = map[string]any{}
	}
	data, err := jsonString(n.Data)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO notifications (user_id, kind, actor_id, offer_id, match_id, round, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())`, n.UserID, n.Kind, nullID(n.ActorID), nullID(n.OfferID), n.MatchID, n.Round, data)
	return err
}

// Notifications lists the newest notifications for the bell.
func (s *Store) Notifications(me int64, limit int) ([]notificationView, error) {
	rows, err := s.db.Query(`SELECT n.id, n.kind, coalesce(a.username, ''), n.match_id, n.round, n.data::text, n.read, n.created_at,
			n.kind = 'friend_request' AND EXISTS (SELECT 1 FROM friend_requests r WHERE r.from_id = n.actor_id AND r.to_id = n.user_id),
			`+offerColumns+`
		FROM notifications n
		LEFT JOIN users a ON a.id = n.actor_id
		LEFT JOIN offers o ON o.id = n.offer_id
		LEFT JOIN users ofrom ON ofrom.id = o.from_id
		LEFT JOIN users oto ON oto.id = o.to_id
		WHERE n.user_id = $1 ORDER BY n.id DESC LIMIT $2`, me, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []notificationView{}
	for rows.Next() {
		var n notificationView
		var data string
		var at time.Time
		var ov offerView
		var oid sql.NullInt64
		var settings, color, status, matchID, ofrom, oto sql.NullString
		var oat sql.NullTime
		if err := rows.Scan(&n.ID, &n.Kind, &n.Actor, &n.MatchID, &n.Round, &data, &n.Read, &at, &n.Pending,
			&oid, &settings, &color, &status, &matchID, &oat, &ofrom, &oto); err != nil {
			return nil, err
		}
		n.Data = json.RawMessage(data)
		n.At = at.UnixMilli()
		ov.Color = color.String
		ov.ID, ov.Status, ov.MatchID, ov.From, ov.To = oid.Int64, status.String, matchID.String, ofrom.String, oto.String
		n.Offer = scanOffer(settings, &ov, oat)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) UnreadNotifications(me int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM notifications WHERE user_id = $1 AND NOT read`, me).Scan(&n)
	return n, err
}

func (s *Store) MarkNotificationsRead(me int64) error {
	_, err := s.db.Exec(`UPDATE notifications SET read = true WHERE user_id = $1 AND NOT read`, me)
	return err
}
