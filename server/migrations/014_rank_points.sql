-- Rank scores replace Glicko-2: both ratings start at 1000 and move by whole
-- points (server/points.go). Everyone moves down by the 500 the starting
-- rating dropped, so the order of the leaderboards stays as it was.
ALTER TABLE users ALTER COLUMN rating SET DEFAULT 1000;
ALTER TABLE users ALTER COLUMN four_rating SET DEFAULT 1000;
UPDATE users SET rating = GREATEST(rating - 500, 0), four_rating = GREATEST(four_rating - 500, 0);

-- The UTC day of each player's last ranked loss in each game; the first loss
-- of a day costs less. deviation and volatility are no longer used.
ALTER TABLE users ADD COLUMN lost_on date;
ALTER TABLE users ADD COLUMN four_lost_on date;

-- Migration 013 gave blueprint (rating:1150) to everyone at the old 1500
-- start. Take it back from those now below 1150 who didn't beat Vercingetorix.
DELETE FROM unlocks u
USING users
WHERE u.reason = 'rating:1150' AND users.id = u.user_id
	AND round(GREATEST(users.rating, users.four_rating)) < 1150
	AND NOT EXISTS (SELECT 1 FROM unlocks v WHERE v.user_id = u.user_id AND v.reason = 'bot:vercingetorix');
