-- A few demo accounts with an avatar and their own piece sets, so friends
-- lists and the leaderboard show more than the default pawn.
UPDATE users SET avatar = v.avatar, prefs = v.prefs::jsonb
FROM (VALUES
	('alice', 'red/Q', '{"primary":"red","secondary":"ink","side":"w"}'),
	('bob', 'green-ink/N', '{"primary":"green-ink","secondary":"cream","side":""}'),
	('carol', 'violet/K', '{"primary":"classic","secondary":"violet","side":"b"}'),
	('dave', 'blue-ink/J', '{"primary":"blue-ink","secondary":"classic","side":""}')
) AS v(username, avatar, prefs)
WHERE lower(users.username) = v.username AND users.avatar = '';
