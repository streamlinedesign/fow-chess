# Changelog

All notable changes, newest first. `make release` turns Unreleased into a version.

## [Unreleased]

- Ancient tiles: a Greek key border on all four sides of each square that joins up with the next square, in place of the single band along the bottom.
- Themed piece sets: galaxy, blueprint, holy and ancient, light for White and dark for Black, for your pieces or your opponent's. Their light and dark pieces are avatars too.
- Tile skins: pick a skin for the light squares and one for the dark squares (galaxy, blueprint, holy, ancient or plain). They cover only your half of the board.
- Skins switch: off draws classic pieces (bots included) and the plain board everywhere, keeping your picks for when you turn it back on.
- Your pieces: Settings has a Pieces and avatar section, for guests too. Pick a set for your own pieces and one for your opponent's (classic, cream, ink, red, blue, green, violet, and the four colours with black lines), and the side you play on the home board and against bots. Signed-in players keep them on the account; guests in the browser. The home board has a Pieces link.
- Avatars: the pawn stays the default, and every piece of every one-colour set can be your avatar. Others see it next to your name in lists, profiles, the bell and Watch.
- Black-lined versions of the red, blue, green and violet sets.
- Bots play with their own coloured sets, on the home board and in bot games: Solomon red, Vercingetorix green, Rex inutilis violet, Ramesses II cream. Against Ramesses as Black, your White pieces are drawn black so the two sides differ.
- Home page board: pawns that reach the last rank ask what to become (queen, rook, bishop or knight) instead of always turning into a queen, and the centre vision squares are ringed and light up as in a real game.
- Bot faces have their own colours: Solomon red, Vercingetorix green, Rex inutilis violet, Ramesses II cream.
- Sync Move for two-player mods games: both players send a move each round without seeing the other's, then both happen at once, with captures last, as in four-player Sync Move. A sent move is locked in. Both clocks run until each side sends. Sync Move games are always Eat the King.
- Backward pawns are gone: the mod, its rule in the engine (two- and four-player) and its strings.
- Dragging a piece on the four-player board no longer selects (and highlights) the squares it passes.
- Four players: Back row and Battleship layouts. Everyone arranges their own three rows at the same time without seeing anyone else's, and the clock starts once all four are ready. Bots shuffle their back row.
- Four players: Sync Move. Everyone sends a move each round without seeing the others', then all the moves happen at once. Pieces leave their squares first and captures come last, so a piece that moves away dodges, two pieces can swap squares, and pieces that land on the same square knock each other out. A sent move is locked in. All clocks run at once; yours stops when you send.
- Mods page: Current setup is part of the Mods section, without a box. The clock picker is the home page's: Short or Long (or No clock on the mods page), then three presets. The extra Bullet, Blitz, Rapid and Classical mods clocks are no longer offered.
- Mods page: what the settings mean is in a Current setup card above the settings, not under the board.
- Bot faces: Rex inutilis is a jester pawn, Ramesses II a prince, Solomon a king.
- The home page no longer has the Live now panel; the top 100 takes the full row. The fallen king in the logo sits 10px higher.
- Home page for the beta: a bigger board with the opponent and the moves beside it, and a bot picker. Solomon is the default bot (the server's demo game now takes any bot). The "How Ramesses plays" panel is gone. Play and New match stand alone with what you've picked written above them. "Mode" is now "Layout". New About the game section: Eat the King is the original game, everything else is a mod, and where the ideas came from.
- The logo has a fallen king before the name and a beta tag after it.
- Mods page: a bigger board and the settings in two columns on wide screens.
- Switches follow the Streamline design system: a square lever, hollow when off and filled when on, instead of the round pill.
- Bug reports: the footer says the game is in beta and has a "Report a bug" link, and game pages have a bug button. A report takes a category (looks, rules, connection, clocks, bots, account, other), the match code (filled in from the game) and what happened. Reports are stored in Postgres; `make bugs` prints the newest.
- Four-player chess under Mods: a cross-shaped board, Red, Blue, Green and Violet moving clockwise, last king standing wins. Works with fog, the hill, the twists, centre vision and any clock; empty seats can be filled with bots. New dusty red, blue, green and violet piece sets.
- The prince (Jump King) now has the king's head under its coronet.
- The Jump King piece is now a prince (a young king with a coronet) instead of a crowned knight.
- Jump King games draw the kings as a crowned knight (`wX.svg`, `bX.svg`), so you can tell at a glance that they jump.
- Jester pawns mod: pawns also step and take one square in any direction, like a king, and keep the double first step, en passant and promotion. Pawns are drawn as jesters in those games. New card on the rules page.
- Centre vision is part of the game: any piece on d5/e5 (White) or d4/e4 (Black) shows its player the whole board while it stays. The squares are ringed on the board. Mods games can turn it off.
- The king-in-danger ring now also shows when the attacker is hidden in the fog (without saying where it is).
- Pawns on their starting rank always see both squares ahead.
- The auto-PR workflow warns instead of failing when the repo doesn't yet let Actions open or auto-merge PRs.
- House conventions: `CLAUDE.md`, `VERSION`, `TODO.md`, CI and auto-PR workflows.
- Server listens on `PORT` (4100) and Vite on `WEB_PORT` (4101) instead of 8080/5173.
- `make db-seed` loads demo accounts; `make deploy`, `make release`, `make configure-repo`.

## [0.1.0] - 2026-11-01

- Everything so far: fog-of-war chess, accounts, ratings, clocks, friends and
  messages, game modes, tournaments, bot opponents, hub, ranked and find match.
