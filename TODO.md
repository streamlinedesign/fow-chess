# Todo

## Now

- Review and merge PR #20 (daily Blitz and Rapid arenas)
- Delete the five stale `claude/project-thread-*` branches once their PRs are closed
- Run `make configure-repo` once so auto-merge and branch protection are on

## Next

- Beta launch: check the new home page and mods page on a phone
- Copy the jester pawns engine change into `chess-bots` (or extract the engine module first)
- Review the mod artwork: jester pawn (`wJ`/`bJ`) and jump king (`wX`/`bX`) in `web/src/pieces/`
- Adopt the Streamline design system (typeface, tokens) in place of Geist/Tailwind defaults

- Extract `server/engine` into a Go module so `chess-bots` stops carrying a copy
- First droplet deploy: unit file and Caddy block in `deploy/`, `make deploy`

## Later

- Four-player: show these games on profiles, Watch and the bell; rematch and draw offers; a stronger bot
- Decide what else the hub should show

## Done

- 0.5.0: bug reports from the footer and game pages; read them with `make bugs`
- 0.1.0: the game, accounts, ratings, modes, tournaments, bots
