# Todo

## Now

- Review and merge PR #20 (daily Blitz and Rapid arenas)
- Delete the five stale `claude/project-thread-*` branches once their PRs are closed
- Run `make configure-repo` once so auto-merge and branch protection are on

## Next

- CI has no Postgres, so every database test skips there (a broken unlock test merged that way); add a Postgres service to the CI workflow
- Security before the beta, still open (from the review): security headers (CSP, frame-ancestors, nosniff, HSTS) in the Caddy block when `deploy/` is written; never run `make db-seed` against production (known `demo` passwords)
- Copy the backward pawns removal (`engine/`) into `chess-bots`
- Copy the jester pawns engine change into `chess-bots` (or extract the engine module first)
- Review the mod artwork: jester pawn (`wJ`/`bJ`) and jump king (`wX`/`bX`) in `web/src/pieces/`
- Adopt the Streamline design system (typeface, tokens) in place of Geist/Tailwind defaults

- Extract `server/engine` into a Go module so `chess-bots` stops carrying a copy
- First droplet deploy: unit file and Caddy block in `deploy/`, `make deploy`

## Later

- Set `BETA_ENDS` in `/etc/<name>/env` when the beta closes, so cyberpunk stays with beta accounts only
- Unlocks: a guest who beats a bot keeps nothing; decide whether to hold the win until they sign up. Replays and Watch don't draw a bot's set or tiles yet
- 1.0: desktop app downloads. Set the `GAME_URL` repository variable (Actions > Variables), `make release`, then turn on `DOWNLOADS` in `web/src/features.ts`
- 1.0: split the rules engine into its own Go repo (`streamlinedesign/chess-engine`) shared by the server and chess-bots
- Desktop app: code signing (Apple Developer account, Windows certificate), a notification when it's your move, `eattheking://` game links, auto-update
- Tournaments are hidden (`TOURNAMENTS` in `web/src/features.ts`); decide when to bring them back, and whether the server should stop scheduling arenas meanwhile
- Four players: a player's skin can share a colour with another seat (violet set next to the violet seat); decide whether to steer away from that, and whether the seat dots should follow the skin
- Skins are kept only in memory for live games: after a server restart a player's skin shows once they reconnect

- Side preference for games shared by code (the creator always plays White for now)

- chess-bots: copy `server/engine` (Sync Move, jester pawns and more are missing there)
- Four-player: drag pieces in the setup (taps only for now); bots that think about Sync Move rounds (they play as if in turns, and bot-only games can go on for a very long time)
- Four-player: show these games on profiles, Watch and the bell; rematch and draw offers; a stronger bot
- Four-player extended board: the seat names and clocks aren't drawn in the corners (they're playable there), only in the list beside the board; find a place for them on the board
- War Mode: bot levels for four players (every bot but Rex inutilis plays `four.BotMove` on the home page board for now); a bot or friendlier wait when fewer than four are in the queue; show War Mode games in the profile's game lists
- Decide what else the hub should show

## Done

- 0.17.0: cyberpunk pieces and tiles for beta players
- 0.16.1: avatar framing, logo on phones, match-not-found crash, knocked-out four-player players kept behind and out of the chat, release date on the home page
- 0.11.0: White and Black sets that opponents see, light and dark lines for colours, tiles on your half (three rows with four players), faster settings, head avatars
- 0.10.1: themed piece sets (galaxy, blueprint, holy with the wing knight, ancient), tile skins with Greek key ancient tiles, skins switch
- 0.5.0: bug reports from the footer and game pages; read them with `make bugs`
- 0.1.0: the game, accounts, ratings, modes, tournaments, bots
