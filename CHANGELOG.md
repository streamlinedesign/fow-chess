# Changelog

All notable changes, newest first. `make release` turns Unreleased into a version.

## [Unreleased]

- Home page: Eat the King and War Mode are marked as the official games, with an Official tag, an ink bar down the side and your rating in each. War Mode's button just says Play.
- Your ratings moved from the header into your menu, both of them: Eat the King and War Mode. `/api/me` returns `fourRating`.
- Settings has a cog icon; Mods has a dial.
- "Inspired by" no longer lists Hungry Hungry Hippos.
- Fixed the unlocks test, which still expected a new account to have no themes after Cyberpunk became everyone's for the beta (main was red).
- Bots: `bots.ThinkTime` caps the time a move takes (600 ms); 0 lifts it, so chess-bots can play a seed the same way every time.
- Stronger bots. One-against-one, the searching bots share a new search (iterative deepening, transposition table, captures played out) that looks only a few plies deep under fog, where a deeper look trusts a board missing every hidden piece. Solomon imagines where the pieces it can't see might be, many times over, and plays the move that does best on average. Bots use the king warning like a player does. In 400-game matches under Eat the King's rules each bot beat its old self: Vercingetorix by about 215 Elo, Ramesses II 110, Solomon 270; the ladder stays in order. Rex inutilis still plays at random. Four players: a new bot that plays Sync Move as it is (a piece is in danger on any square someone could move to this round, a piece that moves away dodges, two pieces landing together both go), with levels: Vercingetorix plays the old bot, Ramesses II and Solomon the new one, Solomon also lining up next round's captures. In War Mode Solomon wins 85% of games against three of the old bot (25% is level). `go test ./bots -run Strength` and `./four -run FourBotStrength` replay the matches.
- Cyberpunk king, queen and prince: the glitched star is now the galaxy's two sparkles in the same place and size, with a neon ghost beside them and a thin cut across, in place of the sliced, outlined star. Ancient pieces: the Greek key on each foot is centred on it (it sat to the left, most on the pawn and jester).
- Back to the default rules: Eat the King warns you when your king is attacked and has centre vision again, and `/rules` is one page. This undoes #45 (in 0.18.0, never released).
- Cyberpunk pieces and tiles, for everyone in the beta: ice white with dark lines for White, deep violet with neon cyan lines for Black, a pink and cyan glitch edge, and a glitched star on the king, queen and prince; flat cyan and violet squares with neon corners. Everyone has them while the beta is open, guests included; after `BETA_ENDS` accounts made before then keep them. Drawn by `scripts/cyberpunk.py`.
- Fixed: opening a match link that no longer exists (a mistyped code, or a game from before a server restart) crashed the page with "Something broke" instead of saying the match wasn't found.
- Security: a player knocked out of a four-player fog game no longer sees the board live (they watch a few moves behind, like spectators), and while the game goes on only other players who are out read what they write.
- Avatars show the whole head of the piece inside the circle, centred, for every kind and set; the menu button's avatar sits in the middle of its ring.
- The fallen king in the logo lines up with the name on phones (its offset now scales with the type size).
- Profile on a phone: each rating keeps its label beside it ("Eat the King rating", "War Mode rating"), and the numbers sit two by two.
- The home page says when the full game ships: 2 February 2027.
- War Mode is 30 minutes each plus 10 seconds a move (was 10 minutes); placing your pieces still has two minutes. The home page board has a War Mode toggle: you against three copies of the bot you pick, starting on your Battleship setup, no clock, nothing saved.
- Themed skins are earned: each bot plays a theme on its own tiles (Rex inutilis holy, Vercingetorix blueprint, Ramesses II ancient, Solomon galaxy), and beating it while signed in unlocks that set, its tiles and its avatars. Plain colours stay free. Locked skins show in Pieces with the bot to beat; the server refuses them in settings and drops them from live games. Themed avatars picked before go back to the pawn. `GET /api/unlocks` lists what you have. A "beta" lock is ready for skins everyone gets during the beta (`BETA_ENDS`).
- The desktop app's downloads are hidden until 1.0: no Download link, menu item or home page strip, and /download shows not found. Switch them back on with `DOWNLOADS` in `web/src/features.ts`. The app can still be installed on a phone from the browser.
- Desktop app for Windows, Mac and Linux (`desktop/`, Rust and Tauri): the game in its own window, with a loading screen while it connects and a "can't reach the game" screen that retries by itself. Its home has three ways to play side by side: Eat the King, War Mode (the ranked four-player game) and Modded, with your rating and place under the two ranked ones. Installers are built for each release tag and attached to the GitHub release.
- The site has a Download link and a /download page (the build for your system, every build, and how to add the site to a phone's home screen), and a strip on the home page on computers.
- The site can be installed on a phone: it has a web app manifest and icons.
- `GET /api/ranks`: your place on both leaderboards and how many players are on each.
- Two official games on the home page, each ranked on its own: Eat the King (one against one, fog, always the usual starting position; Back row and Battleship are only under Mods now) and War Mode (the ranked four-player game: extended board, Battleship, Sync Move, jester pawns, no fog).
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
