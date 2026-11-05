# Todo

## Now

- Review and merge PR #20 (daily Blitz and Rapid arenas)
- Delete the five stale `claude/project-thread-*` branches once their PRs are closed
- Run `make configure-repo` once so auto-merge and branch protection are on

## Next

- Copy the backward pawns removal (`engine/`) into `chess-bots`
- Beta launch: check the new home page and mods page on a phone
- Copy the jester pawns engine change into `chess-bots` (or extract the engine module first)
- Review the mod artwork: jester pawn (`wJ`/`bJ`) and jump king (`wX`/`bX`) in `web/src/pieces/`
- Adopt the Streamline design system (typeface, tokens) in place of Geist/Tailwind defaults

- Extract `server/engine` into a Go module so `chess-bots` stops carrying a copy
- First droplet deploy: unit file and Caddy block in `deploy/`, `make deploy`

## Later

- Four players: a player's skin can share a colour with another seat (violet set next to the violet seat); decide whether to steer away from that, and whether the seat dots should follow the skin
- Skins are kept only in memory for live games: after a server restart a player's skin shows once they reconnect

- Side preference for games shared by code (the creator always plays White for now)

- chess-bots: copy `server/engine` (Sync Move, jester pawns and more are missing there)
- Four-player: drag pieces in the setup (taps only for now); bots that think about Sync Move rounds (they play as if in turns, and bot-only games can go on for a very long time)
- Four-player: show these games on profiles, Watch and the bell; rematch and draw offers; a stronger bot
- Four-player extended board: the seat names and clocks aren't drawn in the corners (they're playable there), only in the list beside the board; find a place for them on the board
- Decide what else the hub should show

## Done

- 0.11.0: White and Black sets that opponents see, light and dark lines for colours, tiles on your half (three rows with four players), faster settings, head avatars
- 0.10.1: themed piece sets (galaxy, blueprint, holy with the wing knight, ancient), tile skins with Greek key ancient tiles, skins switch
- 0.5.0: bug reports from the footer and game pages; read them with `make bugs`
- 0.1.0: the game, accounts, ratings, modes, tournaments, bots
