-- Presence and privacy. last_seen is when the player last had the site open.
ALTER TABLE users
	ADD COLUMN last_seen     timestamptz,
	ADD COLUMN hide_online   boolean NOT NULL DEFAULT false,
	ADD COLUMN offers_from   text NOT NULL DEFAULT 'anyone', -- 'anyone' or 'friends'
	ADD COLUMN requests_from text NOT NULL DEFAULT 'anyone'; -- 'anyone' or 'nobody'

-- A deleted account leaves its games behind, played by a guest.
ALTER TABLE games
	DROP CONSTRAINT games_white_user_fkey,
	ADD CONSTRAINT games_white_user_fkey FOREIGN KEY (white_user) REFERENCES users(id) ON DELETE SET NULL,
	DROP CONSTRAINT games_black_user_fkey,
	ADD CONSTRAINT games_black_user_fkey FOREIGN KEY (black_user) REFERENCES users(id) ON DELETE SET NULL;

-- One row per pair of friends, lower user ID first.
CREATE TABLE friends (
	a          bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	b          bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at timestamptz NOT NULL,
	PRIMARY KEY (a, b),
	CHECK (a < b)
);
CREATE INDEX friends_b ON friends (b);

CREATE TABLE friend_requests (
	from_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	to_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at timestamptz NOT NULL,
	PRIMARY KEY (from_id, to_id)
);
CREATE INDEX friend_requests_to ON friend_requests (to_id);

-- Match offers. Accepting one starts a match between the two.
CREATE TABLE offers (
	id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	from_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	to_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	settings   jsonb NOT NULL,
	color      text NOT NULL DEFAULT 'random', -- the offering player's color: w, b or random
	status     text NOT NULL DEFAULT 'open', -- open, accepted, declined, cancelled
	match_id   text,
	created_at timestamptz NOT NULL
);
CREATE INDEX offers_pair ON offers (from_id, to_id) WHERE status = 'open';

-- Messages between friends, lower user ID first. Kept until both agree to delete the chat.
CREATE TABLE messages (
	id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	a          bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	b          bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	sender     bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	body       text NOT NULL DEFAULT '',
	offer_id   bigint REFERENCES offers(id) ON DELETE CASCADE,
	created_at timestamptz NOT NULL,
	CHECK (a < b)
);
CREATE INDEX messages_pair ON messages (a, b, id);

-- Each side of a chat: the last message read, and whether they asked to delete it.
CREATE TABLE chat_state (
	user_id          bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	other_id         bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	last_read        bigint NOT NULL DEFAULT 0,
	delete_requested boolean NOT NULL DEFAULT false,
	PRIMARY KEY (user_id, other_id)
);

-- What the bell shows: friend requests, match offers, results, and slow moves.
CREATE TABLE notifications (
	id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	kind       text NOT NULL,
	actor_id   bigint REFERENCES users(id) ON DELETE CASCADE,
	offer_id   bigint REFERENCES offers(id) ON DELETE CASCADE,
	match_id   text NOT NULL DEFAULT '',
	round      integer NOT NULL DEFAULT 0,
	data       jsonb NOT NULL DEFAULT '{}',
	read       boolean NOT NULL DEFAULT false,
	created_at timestamptz NOT NULL
);
CREATE INDEX notifications_user ON notifications (user_id, id DESC);
