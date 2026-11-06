-- What each player has unlocked, by reason: "bot:<level>" for beating that
-- bot. A themed skin is locked until its reason is here (see themeLocks in
-- unlocks.go); the plain colours are never locked.
CREATE TABLE unlocks (
	user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	reason     text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (user_id, reason)
);

-- Avatars from themed sets were free until now; they go back to the pawn.
-- (Themed piece sets and tiles fall back to the default when read.)
UPDATE users SET avatar = ''
WHERE split_part(avatar, '/', 1) ~ '^(galaxy|blueprint|holy|ancient)-';
