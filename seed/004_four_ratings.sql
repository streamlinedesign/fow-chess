-- Four-player ratings for the demo opponents, so the four-player leaderboard
-- has rows. demo hasn't played a ranked four-player game yet.
UPDATE users SET four_rating = v.rating, four_deviation = 150, four_rated_games = v.games
FROM (VALUES
	('alice', 1110, 12),
	('bob', 990, 7),
	('carol', 1055, 9),
	('dave', 920, 5)
) AS v(username, rating, games)
WHERE lower(users.username) = v.username;
