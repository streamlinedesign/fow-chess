# Chess

Two-player online chess. One player creates a match and shares the 6-character ID (or link), and the other joins with it.

- **web/**: React 19, Vite, TypeScript, Tailwind v4, Radix, Geist. The browser knows no chess rules: it draws the board the server sends and only offers the legal moves the server lists.
- **server/**: Go, with Postgres for accounts and games. It holds every live match in memory, checks each move with its own rules engine, and pushes state over WebSockets. In production it also serves the built web app from inside the binary.
- **server/engine/**: the rules engine. Positions, FEN, move generation, legality, SAN, and game results. It has no dependencies and knows nothing about networking, so variants like fog of war build on top of it.

Rules: full legal moves including castling, en passant and promotion (with a picker), check, checkmate, stalemate, insufficient material, threefold repetition and the fifty-move rule (both end the game automatically). Players can offer a draw, resign, and rematch (colors swap). Refreshing or losing signal reconnects you to your seat. Anyone else who opens the link watches.

## Fog of war and match settings

The creator picks the rules when making a match, for play testing:

- **Fog of war** on or off. With fog on, each piece sees its own square and every square it could move to, captures included. A pawn always sees the square straight ahead, even when blocked, and sees its diagonals only when it can capture there. You get no hints about the opponent: no last-move highlight, `?` in the move list for their moves, and no check marks on your own moves. Spectators see only the last move of each side. When the game ends, everyone sees the full board.
- **Win by** checkmate (normal check rules) or Eat the King (win by capturing the king). In Eat the King mode there is no check: kings may walk into danger and may castle out of, through or into attack.

The rules live in `server/engine` (`Rules`, `Position.Vision`), and what each viewer receives is decided in `server/view.go`.

## Accounts and saved games

Players can sign up with just a username and password (no email). Passwords are hashed with bcrypt, and sessions are a random token in an http-only cookie, with only a hash of it stored. Guests can still play without an account.

Everything lives in Postgres. The server connects to `DATABASE_URL` (or `-db <url>`), and defaults to the local Postgres from `docker-compose.yml`. Tables are created and updated on startup from the numbered SQL files in `server/migrations/`: to change the schema, add a new file rather than editing one that has shipped.

- Every game is saved as it's played, so a server restart no longer ends games: reopen the link and play on.
- Games with at least one signed-in player are kept forever. Games between two guests are deleted after 30 days.
- A guest who signs in mid-game keeps their seat.
- `/u/<username>` shows a player's rating, current games and finished games. Finished games can be replayed move by move at `/g/<match id>/<round>`, with the fog lifted. Games still being played can't be replayed, since that would reveal the board.
- Everyone starts at a rating of 1500. One rating covers every game mode. It starts changing once rated games arrive with clocks.

## Chat

The two players can chat during a game. Spectators don't see it. Messages are kept only in the server's memory, never in the database: a rematch starts a fresh chat, and once a game is over and both players have left, its chat is deleted. A server restart clears every chat. Messages are capped at 500 characters, and a game keeps its last 200.

## Landing page replay

The landing page replays Morphy's Opera Game as White and Black each saw it under fog. The frames in `web/src/showcase.json` are produced by the server's own rules, and a test fails if they drift. After changing what players see, regenerate them with `cd server && go test -run Showcase -update`.

## Requirements

- Go 1.24+
- Node 20+
- Docker (Docker Desktop on a Mac), which runs Postgres for you

## Run locally

Start Docker Desktop, then:

```sh
make dev
```

Then open http://localhost:5173. `make dev` starts Postgres in Docker first (the first run downloads the image, about 30 seconds), waits until it's ready, and the server creates its tables. Postgres keeps running in the background after you stop `make dev`, so the next start is instant.

- `make db-stop` stops Postgres. Your accounts and games are kept.
- `make db-reset` deletes every account and game and starts fresh.
- `make psql` opens a SQL shell on the database.
- Already running your own Postgres on port 5432? Stop it, or create a `chess` database there and set `DATABASE_URL` to it: `make dev` then skips Docker and uses yours.
- Accounts and games from the old SQLite `chess.db` aren't carried over. To play against yourself, open a second browser or a private window, since each browser gets one seat. Vite is started with `--host`, so a phone on the same Wi-Fi can open `http://<your-computer-ip>:5173`.

## Tests

```sh
make test                          # starts Postgres if needed, then runs every Go test
cd server && go test ./...         # includes perft to depth 4-5 on the standard test positions (~3s)
cd server && go test -short ./...  # skips the deep perft runs
```

Tests that touch the database each get their own Postgres schema, dropped afterwards, so they never touch your dev data. They use `CHESS_TEST_DATABASE_URL` if set, else the Docker Postgres, and are skipped with a note if neither is running.

If a page says **Server out of date** or `make dev` says port 8080 is in use, an older copy of the server (often a previously built `./chess`) is still running. Stop it and run `make dev` again.

## Production build

```sh
make build      # builds web, embeds it, writes ./chess
DATABASE_URL=postgres://user:pass@host:5432/chess ./chess   # serves everything on :8080 (use -addr :3000 to change)
```

`make build-linux` also writes `chess-linux-amd64` for a droplet.

## Droplet (later)

1. Copy `chess-linux-amd64` to the droplet, for example `/opt/chess/chess`.
2. Run it with systemd:

   ```ini
   # /etc/systemd/system/chess.service
   [Unit]
   Description=chess
   After=network.target

   [Service]
   Environment=DATABASE_URL=postgres://chess:CHANGE_ME@localhost:5432/chess?sslmode=disable
   ExecStart=/opt/chess/chess -addr 127.0.0.1:8080
   Restart=always
   DynamicUser=yes

   [Install]
   WantedBy=multi-user.target
   ```

3. Put Caddy in front for HTTPS and compression:

   ```
   chess.example.com {
       encode zstd gzip
       reverse_proxy 127.0.0.1:8080
   }
   ```

Accounts and games live in Postgres (install it on the droplet or use a managed one), which survives redeploys; back it up with `pg_dump`. Idle matches leave memory after 2 hours and load again from the database when someone opens them.

## Protocol

- `POST /api/matches?player=<browser id>` with an optional body `{"fog":true,"kingCapture":true}` returns `{"id":"K7QF2M"}`. The creator gets white.
- `GET /api/matches/{id}` returns 204 if the match exists, 404 if it doesn't
- `GET /ws/{id}?player=<browser id>`: the server sends `{"type":"state", ...}` to each viewer after every change, built for that viewer:
  - `board`: 64 characters from a1 to h8. FEN letters for pieces (uppercase is white), `.` for empty, and `?` for squares the viewer can't see (reserved for fog of war).
  - `settings`, `turn`, `ply`, `last` (`{from,to}`), `check` (king square), `history` (SAN, `?` for moves hidden by fog).
  - `players`: `{"w":{"taken":true,"online":true,"name":"Uriel"},"b":{...}}`, where `name` is the username of a signed-in player and absent for a guest.
  - `captured`: `{"w":"qnp","b":"P"}`, the pieces each side has taken as FEN letters, most valuable first. Sent to everyone, since players know this even under fog.
  - `legal`: UCI moves the viewer may play now, sent only to the player whose turn it is.
- The client sends `{"type":"move","uci":"e7e8q"}`, `{"type":"draw"}`, `{"type":"resign"}`, `{"type":"rematch"}`, or `{"type":"ping","t":<ms>}`, which the server answers with `{"type":"pong","t":<same>}`. The draw and rematch messages toggle the offer.
- Players send `{"type":"chat","text":"gg"}`. The server sends players `{"type":"chat","reset":true,"lines":[...]}` with the whole chat when they connect and when a rematch clears it, then `{"type":"chat","lines":[{"id":3,"from":"w","text":"gg","at":<unix ms>}]}` for each new message.
- A slow connection skips straight to the newest state instead of queueing old ones. Chat messages and pongs are never skipped.

## Measuring sync speed

The match page shows `ping` (round trip to the server) and how long the server took to confirm your last move. Run the server with `./chess -v`, or `cd server && go run . -v`, to log every action with how long it took to handle (about 0.1 ms per move).

## Planned

Setup modes, to be chosen in the match settings:
- **Default pieces**: the normal starting position (what exists today).
- **Set your back row**: each player arranges their own back-rank pieces before the game.
- **Set all pieces**: each player places all of their pieces before the game.

## Credits

Piece set "cburnett" by Colin M.L. Burnett, CC BY-SA 3.0.
