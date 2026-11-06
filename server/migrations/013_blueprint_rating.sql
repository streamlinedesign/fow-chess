-- Vercingetorix now wears Survey, and blueprint is earned by reaching a 1150
-- rating (ratingUnlocks in unlocks.go). Players who beat Vercingetorix keep
-- blueprint, and so does everyone whose rating is at 1150 now, so it stays if
-- their rating falls later.
INSERT INTO unlocks (user_id, reason, created_at)
SELECT user_id, 'rating:1150', now() FROM unlocks WHERE reason = 'bot:vercingetorix'
UNION
SELECT id, 'rating:1150', now() FROM users WHERE round(GREATEST(rating, four_rating)) >= 1150
ON CONFLICT DO NOTHING;
