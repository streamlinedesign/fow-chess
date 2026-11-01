-- Arenas: scheduled tournaments where players are paired as soon as they're
-- free, scoring 2 for a win and 1 for a draw until the clock runs out.
CREATE TABLE tournaments (
	id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	kind       text NOT NULL, -- blitz or rapid
	starts_at  timestamptz NOT NULL,
	ends_at    timestamptz NOT NULL,
	status     text NOT NULL DEFAULT 'scheduled', -- scheduled, running, finished, cancelled
	UNIQUE (kind, starts_at)
);

CREATE TABLE tournament_players (
	tournament_id bigint NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
	user_id       bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	score         int NOT NULL DEFAULT 0,
	games         int NOT NULL DEFAULT 0,
	wins          int NOT NULL DEFAULT 0,
	paused        boolean NOT NULL DEFAULT false, -- sitting out: no new pairings
	joined_at     timestamptz NOT NULL,
	PRIMARY KEY (tournament_id, user_id)
);

CREATE TABLE tournament_games (
	tournament_id bigint NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
	match_id      text NOT NULL,
	round         int NOT NULL,
	white_user    bigint NOT NULL,
	black_user    bigint NOT NULL,
	result        text NOT NULL DEFAULT '', -- 1-0, 0-1, 1/2-1/2, or '' while playing
	created_at    timestamptz NOT NULL,
	PRIMARY KEY (match_id, round)
);
CREATE INDEX tournament_games_tournament ON tournament_games (tournament_id);
