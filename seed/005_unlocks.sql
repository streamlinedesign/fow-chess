-- What the demo opponents have unlocked, so their themed skins from
-- 003_avatars.sql show. demo has beaten Vercingetorix (survey), carol has
-- reached a 2000 rating once (holy), and no one has everything, so the locked
-- skins can be seen too.
INSERT INTO unlocks (user_id, reason)
SELECT users.id, v.reason
FROM (VALUES
	('demo', 'bot:vercingetorix'),
	('bob', 'bot:solomon'),
	('carol', 'rating:2000'),
	('dave', 'bot:ramesses')
) AS v(username, reason)
JOIN users ON lower(users.username) = v.username
ON CONFLICT DO NOTHING;
