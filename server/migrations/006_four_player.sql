-- Four-player games from the mods page. One row per match: the moves are
-- replayed to rebuild a game after a restart, like games.
CREATE TABLE four_games (
	match_id        text PRIMARY KEY,
	settings        jsonb NOT NULL,
	seats           jsonb NOT NULL, -- four player keys, red blue green violet; "" for a free seat
	users           jsonb NOT NULL, -- signed-in players' ids, 0 for guests and bots
	names           jsonb NOT NULL, -- usernames, or the bot's name
	moves           text NOT NULL DEFAULT '', -- moves and leavings; see four.Event.Token
	result          text NOT NULL DEFAULT '', -- r, b, g, v, draw, or '' while playing
	reason          text NOT NULL DEFAULT '',
	clock_ms        jsonb NOT NULL, -- each seat's time left as of turn_started_at
	turn_started_at timestamptz,
	created_at      timestamptz NOT NULL,
	updated_at      timestamptz NOT NULL
);
