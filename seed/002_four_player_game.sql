-- A finished four-player game, so /f/DEMO4P shows the board at once:
-- demo (Red), alice (Blue, the winner), bob (Green) and carol (Violet).
-- Safe to run twice. four_match_test.go replays these moves.
INSERT INTO four_games (match_id, settings, seats, users, names, moves, result, reason, clock_ms, created_at, updated_at)
SELECT 'DEMO4P',
	'{"fog":true,"time":{"base":600,"inc":0,"perMove":0},"centerVision":"any"}',
	json_build_array('u:' || d.id, 'u:' || a.id, 'u:' || b.id, 'u:' || c.id),
	json_build_array(d.id, a.id, b.id, c.id),
	'["demo","alice","bob","carol"]',
	'g2g4 b10d10 j13j11 m4k4 f1m8 a9e13 f14e13 n10m8 j1i3 b11d11 i14m10 n9m10 i3k4 a5c6 e13m5 n4k4 e1f3 b9d9 m5n6 n7n6 d2d4 c6d4 e14f12 m10g4 d1d4 a10c9 f12d11 g4f3 e2f3 a11d11 k13k11 k4d4 g1g13 d11j11 h14g13 d4d9 k2k4 j11j14 k14j14 d9c9 j2j4 b8c9 j14j4 n6i1 k1i1 a8m8 g13d10 n8m8 h2h4 c9d10 j4h4 m9k9 i2i4 b6d6 h4h1 n5l6 b7d7 h1h8 m7k7 a6h13 h8m8 h13g14',
	'b', 'capturing the king',
	'[540000,552000,561000,548000]',
	now() - interval '1 day', now() - interval '1 day'
FROM users d, users a, users b, users c
WHERE d.username = 'demo' AND a.username = 'alice' AND b.username = 'bob' AND c.username = 'carol'
ON CONFLICT (match_id) DO NOTHING;
