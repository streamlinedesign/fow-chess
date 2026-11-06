-- A few demo accounts with an avatar and their own piece sets, so friends
-- lists and the leaderboard show more than the default pawn.
UPDATE users SET avatar = v.avatar, prefs = v.prefs::jsonb
FROM (VALUES
	('alice', 'red-light/Q', '{"white":"red","black":"red","side":"w","lightTiles":"","darkTiles":""}'),
	('bob', 'green-ink/N', '{"white":"galaxy","black":"green","side":"","lightTiles":"galaxy","darkTiles":"galaxy"}'),
	('carol', 'violet-ink/K', '{"white":"classic","black":"violet","side":"b","lightTiles":"","darkTiles":"holy"}'),
	('dave', 'ancient-light/J', '{"white":"ancient","black":"blue","side":"","lightTiles":"ancient","darkTiles":"ancient"}')
) AS v(username, avatar, prefs)
WHERE lower(users.username) = v.username AND users.avatar = '';
