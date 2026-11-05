-- Four-player games with Back row or Battleship start from each player's own
-- arrangement. start holds them: {"setups": [four 24-letter strings, "" until
-- placed], "quit": [who resigned while arranging]}.
ALTER TABLE four_games ADD COLUMN start jsonb NOT NULL DEFAULT '{}';
