# Changelog

All notable changes, newest first. `make release` turns Unreleased into a version.

## [Unreleased]

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
