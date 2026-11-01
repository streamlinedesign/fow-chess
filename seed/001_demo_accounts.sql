-- Demo accounts for a fresh local database: demo / demo, and a few opponents
-- with spread-out ratings so the ladder and matchmaking look alive.
-- Loaded by `make db-seed`; safe to run twice.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO users (username, password_hash, rating, deviation, created_at)
SELECT v.username, convert_to(crypt('demo', gen_salt('bf', 10)), 'UTF8'), v.rating, 120, now()
FROM (VALUES
	('demo', 1500),
	('alice', 1720),
	('bob', 1380),
	('carol', 1950),
	('dave', 1100)
) AS v(username, rating)
WHERE NOT EXISTS (SELECT 1 FROM users u WHERE lower(u.username) = v.username);
