-- Bots the demo opponents have beaten, so their themed skins from
-- 003_avatars.sql show. demo has beaten Vercingetorix (blueprint) and no one
-- else yet, so the locked skins can be seen too.
INSERT INTO unlocks (user_id, reason)
SELECT users.id, v.reason
FROM (VALUES
	('demo', 'bot:vercingetorix'),
	('bob', 'bot:solomon'),
	('carol', 'bot:rex'),
	('dave', 'bot:ramesses')
) AS v(username, reason)
JOIN users ON lower(users.username) = v.username
ON CONFLICT DO NOTHING;
