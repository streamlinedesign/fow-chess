-- Clocks. Each side's time left as of turn_started_at, when the side to move's
-- clock last started. deadline is when that side runs out; it's empty while no
-- clock is running, so finding games that ran out of time is one index scan.
ALTER TABLE games
	ADD COLUMN white_clock_ms  bigint,
	ADD COLUMN black_clock_ms  bigint,
	ADD COLUMN turn_started_at timestamptz,
	ADD COLUMN deadline        timestamptz;
CREATE INDEX games_deadline ON games (deadline) WHERE deadline IS NOT NULL;

-- Rating. rated is decided when the game is saved: both players signed in and a
-- clock. The diffs are filled in once the ratings have been updated, which
-- also marks the game as done so it's never rated twice.
ALTER TABLE games
	ADD COLUMN rated             boolean NOT NULL DEFAULT false,
	ADD COLUMN white_rating_diff integer,
	ADD COLUMN black_rating_diff integer;

ALTER TABLE users ADD COLUMN rated_games integer NOT NULL DEFAULT 0;
CREATE INDEX users_rating ON users (rating DESC) WHERE rated_games > 0;
