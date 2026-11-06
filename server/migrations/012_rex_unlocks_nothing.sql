-- Rex inutilis no longer wears or unlocks a theme; holy is earned by reaching
-- a 2000 rating instead (lockRating in unlocks.go). Players who already beat
-- Rex keep holy.
UPDATE unlocks SET reason = 'rating:2000' WHERE reason = 'bot:rex';
