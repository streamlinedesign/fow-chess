CREATE TABLE users (
	id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	username      text NOT NULL,
	password_hash bytea NOT NULL,
	-- Glicko-2, one rating across every game mode.
	rating        double precision NOT NULL DEFAULT 1500,
	deviation     double precision NOT NULL DEFAULT 350,
	volatility    double precision NOT NULL DEFAULT 0.06,
	created_at    timestamptz NOT NULL
);
-- Usernames are unique regardless of case.
CREATE UNIQUE INDEX users_username ON users (lower(username));

CREATE TABLE sessions (
	token_hash bytea PRIMARY KEY,
	user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at timestamptz NOT NULL
);

-- One row per game: a match's rematches are rounds 1, 2, 3...
CREATE TABLE games (
	match_id   text NOT NULL,
	round      integer NOT NULL,
	settings   jsonb NOT NULL,
	white      text NOT NULL DEFAULT '', -- seat key: "u:<id>" or a guest browser ID
	black      text NOT NULL DEFAULT '',
	white_user bigint REFERENCES users(id),
	black_user bigint REFERENCES users(id),
	moves      text NOT NULL DEFAULT '', -- UCI, space separated
	result     text NOT NULL DEFAULT '', -- empty while the game is on
	reason     text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL,
	PRIMARY KEY (match_id, round)
);
CREATE INDEX games_white_user ON games (white_user, updated_at);
CREATE INDEX games_black_user ON games (black_user, updated_at);
