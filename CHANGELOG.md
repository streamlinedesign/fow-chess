# Changelog

All notable changes, newest first. `make release` turns Unreleased into a version.

## [Unreleased]

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
