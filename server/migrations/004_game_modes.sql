-- Setup modes: the starting position of a game where the players arranged
-- their own pieces. Empty for games from the usual starting position.
ALTER TABLE games ADD COLUMN start_fen text NOT NULL DEFAULT '';
