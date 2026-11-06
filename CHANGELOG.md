# Changelog

All notable changes, newest first. `make release` turns Unreleased into a version.

## [Unreleased]

- Tournaments are hidden for now: no links in the header, the menu or the home page, and /tournaments shows not found. Switch them back on with `TOURNAMENTS` in `web/src/features.ts`.
- Ranked four-player games are played with jester pawns, rules and pieces.
- Ranked four-player games: one fixed game (extended board, Battleship, Sync Move, no fog, 10 minutes each), found from the home page in a queue of four by rating. It has its own Glicko-2 rating and its own leaderboard tab; each game counts as a result against each of the other three by finishing order. Players have two minutes to place their pieces, then get a bot's arrangement.
- Four players, extended board: pawns promote past the middle, as on the cross (the ninth rank from their side on), or on the side walls where the players beside them sit.
- Four players, extended board: pawns promote on any edge but their own side (the side edges count past your own rows), not just the far edge. Fixed the board jumping while a move was being sent on the extended board.
- Security: a guest ID can no longer name a signed-in player's or a bot's seat, which let anyone play, resign or see the fog view of any signed-in player's game. Rate limits now count the real address behind Caddy, so a few wrong passwords no longer lock everyone out of signing in. New limits per address on sign-ups, new matches and match codes that don't exist; bug reports count per address whatever browser ID they send.
- Four players: an extended board under Mods, 16 by 16 with four rows each and neutral corners anyone can cross. Centre vision comes from your two centre squares on the far side, pawns promote on the far edge, Battleship uses all four rows, and there's no hill.
- Your look is yours to show: you pick a set for when you play White and one for when you play Black, and your opponent and spectators see your pieces and tiles as you picked them. Watch and replays show them too. Same set on both sides is fine: colours have light lines as White and dark lines as Black, themes their light and dark halves.
- Coloured sets now come with light lines and with dark lines instead of mixed outlines; bots play theirs the same way.
- Tiles cover your whole half of the board in a two-player game, and your own rows (three, or four on the extended board) with four players.
- No skins means classic pieces and the plain board for every player on your screen; bots and the four-player colours stay.
- Four players: a player with a skin plays in one of their two sets, picked at random per game; everyone else keeps their seat colour.
- Settings opens about six times faster on a phone: plain radio buttons, and the avatar is picked by set, then piece.
- Avatars are zoomed on the head of the piece.
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
