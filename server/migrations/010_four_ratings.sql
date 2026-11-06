-- Ranked four-player games have a Glicko-2 rating of their own, apart from
-- the one-against-one rating.
ALTER TABLE users ADD COLUMN four_rating double precision NOT NULL DEFAULT 1500;
ALTER TABLE users ADD COLUMN four_deviation double precision NOT NULL DEFAULT 350;
ALTER TABLE users ADD COLUMN four_volatility double precision NOT NULL DEFAULT 0.06;
ALTER TABLE users ADD COLUMN four_rated_games integer NOT NULL DEFAULT 0;
CREATE INDEX users_four_rating ON users (four_rating DESC) WHERE four_rated_games > 0;

-- How each seat's four-player rating moved, red blue green violet; null
-- until a ranked game has been rated, and for unranked games.
ALTER TABLE four_games ADD COLUMN rating_diff jsonb;
