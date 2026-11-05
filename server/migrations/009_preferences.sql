-- Each player's look: the avatar others see next to their name ("" is the
-- pawn everyone starts with, else "<set>/<piece>" like "red/N"), and their
-- own preferences: {"primary": set, "secondary": set, "side": "w"|"b"|""}.
ALTER TABLE users ADD COLUMN avatar text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN prefs jsonb NOT NULL DEFAULT '{}';
