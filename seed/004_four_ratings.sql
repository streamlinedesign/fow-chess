-- Four-player ratings for the demo opponents, so the four-player leaderboard
-- has rows. demo hasn't played a ranked four-player game yet.
UPDATE users SET four_rating = v.rating, four_deviation = 150, four_rated_games = v.games
FROM (VALUES
	('alice', 1610, 12),
	('bob', 1490, 7),
	('carol', 1555, 9),
	('dave', 1420, 5)
) AS v(username, rating, games)
WHERE lower(users.username) = v.username;
