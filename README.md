# Eat the King

Two-player online chess with fog of war. One player creates a match and shares the 6-character ID (or link), and the other joins with it.

- **web/**: React 19, Vite, TypeScript, Tailwind v4, Radix, Geist. The browser knows no chess rules: it draws the board the server sends and only offers the legal moves the server lists.
- **server/**: Go, with Postgres for accounts and games. It holds every live match in memory, checks each move with its own rules engine, and pushes state over WebSockets. In production it also serves the built web app from inside the binary.
- **server/engine/**: the rules engine. Positions, FEN, move generation, legality, SAN, and game results. It has no dependencies and knows nothing about networking, so variants like fog of war build on top of it.

The site plays one game, **Eat the King**: chess with fog of war, where you win by capturing the king. Full legal moves including castling, en passant and promotion (with a picker), stalemate, insufficient material, threefold repetition and the fifty-move rule (both end the game automatically). Players can offer a draw, resign, and rematch (colors swap). Refreshing or losing signal reconnects you to your seat. Anyone else who opens the link watches.

## The game

Every new match has fog of war and Eat the King on. The server forces both (`normalize` in `server/view.go`), so a client can't ask for anything else.

- **Fog of war**: each piece sees its own square and every square it could move to, captures included. A pawn always sees the square straight ahead, even when blocked, and sees its diagonals only when it can capture there. You get no hints about the opponent: no last-move highlight and `?` in the move list for their moves. Spectators see one side's view, 6 moves behind (see Watching below). When the game ends, everyone sees the full board.
- **Eat the King**: there is no check and no checkmate. Kings may walk into danger and may castle out of, through or into attack. You win by capturing the king.
- **King in danger**: the one warning. When a piece the player can see attacks their king, the king gets a red ring and the match page says so; while the king is picked up, the destination squares a seen piece covers are marked too. It's computed from the player's own view only (`kingDanger` in `server/view.go`): hidden pieces count as blockers, never as attackers, so it leaks nothing. The warning is sent as `danger` and `unsafe` in the game view. When the king is taken, the fog lifts and the capturing move shows for a moment before the game-over dialog.
- **Modes** decide where the pieces start: **Standard** (the usual position), **Back row** (each player arranges their back row; pawns stay put) or **Battleship** (each player places all sixteen pieces anywhere in their own four rows). Both players set up at the same time, and the clock doesn't run until both are ready. You never see the other setup. Castling is allowed only for a king and rook that start on their usual squares. Setups are checked in `server/setup.go`, and a game saves the position it started from (`start_fen`) so replays and restarts work.
- **Ranked or Unranked**: a game can be ranked only with two signed-in players; the creator can turn rating off.
- **Find match** (`POST /api/queue`, `server/queue.go`): a signed-in player waits for the player nearest their rating who wants the same clock and mode. The rating window starts at 100 points and widens by 20 a second, so after half a minute anyone wanting the same game will do. The page polls `GET /api/queue` every 3 seconds, which is also what keeps a player in the queue; the pair is told over live events (`match`). Colours are random.
- **Mods** (`/mods`): the game with every dial exposed: fog off, checkmate rules, King of the Hill, Jump King, backward pawns, and the extra clocks including No clock (`modTimeControls`). A mods game carries `mods: true`, keeps its options through `normalize`, and is never ranked.

A king that starts in danger after setup is left as it is. A win on the winner's very first move is a **Sniper Win**: it counts, but moves both ratings only a quarter as far as a whole game (`sniperWeight` in `server/setup.go`).

The older options (checkmate rules, no fog, King of the Hill, Jump King, backward pawns: `Rules` in `server/engine`) live on as mods, and games saved with them before the site became one game keep working and replaying. They're described with the `legacy` labels in the dictionaries.

The rules live in `server/engine` (`Rules`, `Position.Vision`), and what each viewer receives is decided in `server/view.go`.

## Accounts and saved games

Players can sign up with just a username and password (no email). Passwords are hashed with bcrypt, and sessions are a random token in an http-only cookie, with only a hash of it stored. Guests can still play without an account.

Everything lives in Postgres. The server connects to `DATABASE_URL` (or `-db <url>`), and defaults to a database called `chess` in the Postgres installed on your machine. On startup it creates that database if it's missing, then creates and updates the tables from the numbered SQL files in `server/migrations/`: to change the schema, add a new file rather than editing one that has shipped.

- Every game is saved as it's played, so a server restart no longer ends games: reopen the link and play on.
- Games with at least one signed-in player are kept forever. Games between two guests are deleted after 30 days.
- A guest who signs in mid-game keeps their seat.
- `/u/<username>` shows a player's rating, current games and finished games. Finished games can be replayed move by move at `/g/<match id>/<round>`, with the fog lifted. Games still being played can't be replayed, since that would reveal the board.
- The home page is the game's hub: the match settings and your view of the board, then your games in progress, friends online, a few live games and the Top 100 (`GET /api/leaderboard`, which also returns your own rank when you're signed in). Guests see a prompt to sign in instead of friends. The rules and the Opera Game replay are at `/rules`.

## Clocks and rating

The creator picks a time control from fixed presets (`timeControls` in `server/view.go`, `TIME_GROUPS` in `web/src/settings.ts`, which must match): Blitz 5+0 and 3+2, Rapid 10+0, Tournament 90+30, Daily (24 hours per move, reset after every move) or Weekly (7 days per move). The server refuses anything else for a new game; games saved with an earlier preset, including No clock, keep their clock and still play and replay.

- Clocks run on the server. White's clock starts once Black sits down. The increment is added after each move.
- When a clock runs out, that side loses on time, unless the opponent has only a king, or a king and one minor piece, which is a draw. A timer ends the game even with nobody connected, and every 30 seconds the server also checks the database for games whose deadline passed while it wasn't holding them (after a restart, for example).
- The browser counts down locally from the last state the server sent, so the clock stays smooth between messages.
- A game is **ranked** (rated) when both players are signed in, it isn't a mods game, it has a clock (every preset on offer has one), and the creator didn't make it Unrated. Ratings use Glicko-2, starting at 1500 with one rating for every mode, and each game is its own rating period. The change is applied once, in a transaction, when the finished game is saved, and shown in the game-over dialog and on the profile.

## Chat

The two players can chat during a game. Spectators don't see it. Messages are kept only in the server's memory, never in the database: a rematch starts a fresh chat, and once a game is over and both players have left, its chat is deleted. A server restart clears every chat. Messages are capped at 500 characters, and a game keeps its last 200.

## Friends, messages and the bell

Signed-in players can be friends, message each other and offer matches. Guests keep the code system and none of this.

- **Friends** are mutual: one sends a request, the other accepts it (from the bell, the Friends page or the profile). `/friends` has search by username, your friends with online ones first, and the requests you've sent.
- **Messages** (`/messages`, `/messages/<name>`) are only between friends. They're kept in Postgres until both people ask to delete the chat. A chat also shows match offers as cards, finished games between you as result cards, and your score against each other. The in-game chat on the match page is separate and still kept only in memory.
- **Match offers** come from a profile, the friends list or a chat. The sender picks the rules, the clock and their colour. Between friends the offer also shows up as a card in their chat. Accepting starts a match with both seats taken, and its clock starts once both players have it open.
- **The bell** holds friend requests, match offers and answers to them, game results you weren't there to see, and moves in Daily or Weekly games that took more than 4 hours (`slowMove` in `server/match.go`). It's a dropdown, or a sheet on phones.
- **Live updates** go to every open tab of a signed-in player over server-sent events (`GET /api/live`, `server/live.go`). Having a tab open is what makes a player online.
- **Settings** (`/settings`): theme, change password (which signs out other devices), log out everywhere, delete account, and privacy: hide your online status, take match offers from friends only, and turn off friend requests. A deleted account's games stay, played by a guest.

## Watching

- **Watch** (`/watch`, in the top bar when signed out and in the avatar menu when signed in) lists the games being played right now, with friends' games first, then the strongest players. A game is listed once both seats are taken and a player has it open; before the first move, both players need to be there. Your own games aren't listed. The list refreshes every 10 seconds.
- **Spectators** open a match by its link like anyone else and pick whether to see it as White or Black. In a game still being played, they see exactly what that player saw 6 moves ago (`spectatorDelay` in `server/view.go`, counting both sides' moves), so nobody watching can tell a player what the fog hides. Clocks and whose move it is stay live. Once the game is over, spectators see everything as it is. Players see how many people are watching.
- **In a game**: a friend playing right now shows as "In a game" on the Friends page, their profile and your chat, with a Watch button. Hiding your online status hides this too.

## Rules page replay

The rules page (`/rules`) replays Morphy's Opera Game as White and Black each saw it under fog. The frames in `web/src/showcase.json` are produced by the server's own rules, and a test fails if they drift. After changing what players see, regenerate them with `cd server && go test -run Showcase -update`.

## Languages

The app speaks English and Croatian. It opens in the browser's language when
we have it, and the switch (EN/HR in the top bar for guests, a row in the
avatar menu and in Settings for players) remembers your choice on that
device. Every string the app shows lives in `web/src/i18n/<lang>.ts`; the
English file is the master, and TypeScript makes a missing or misspelled
Croatian key a build error. Server errors come with a `code` that the client
looks up in the dictionary, so they show in the user's language too.

To add a language: copy `web/src/i18n/en.ts` to `<code>.ts`, translate it
(the type `Dict` flags anything you missed), import it in `web/src/i18n/index.ts`
and add a row to `LANGS`. Nothing else changes.

## Requirements

- Go 1.24+
- Node 20+
- PostgreSQL 14+, installed and running (for example `brew install postgresql@17 && brew services start postgresql@17`)

## Run locally

With Postgres running:

```sh
make dev
```

Then open http://localhost:5173. On the first start the server creates a `chess` database and its tables, so there's nothing to set up by hand. It connects through Postgres's local socket as your own user, which is how Homebrew and Postgres.app install it. If your Postgres needs a user, password or another host, set `DATABASE_URL` instead, for example `export DATABASE_URL=postgres://chess:secret@localhost:5432/chess`.

- `make migrate` creates the database if needed and applies migrations, without starting the server.
- `make psql` opens a SQL shell on the database.
- `make db-reset` deletes every account and game in the local `chess` database and starts fresh.
- On Linux, where Postgres only knows the `postgres` user at first, give yourself a role once: `sudo -u postgres createuser --superuser $USER`.
- Accounts and games from the old SQLite `chess.db` aren't carried over.

To play against yourself, open a second browser or a private window, since each browser gets one seat. Vite is started with `--host`, so a phone on the same Wi-Fi can open `http://<your-computer-ip>:5173`.

## Tests

```sh
make test                          # runs every Go test
cd server && go test ./...         # includes perft to depth 4-5 on the standard test positions (~3s)
cd server && go test -short ./...  # skips the deep perft runs
```

Tests that touch the database each get their own Postgres schema, dropped afterwards, so they never touch your dev data. They use `CHESS_TEST_DATABASE_URL` if set, else the same database as the server, and are skipped with a note if Postgres isn't running.

If a page says **Server out of date** or `make dev` says port 8080 is in use, an older copy of the server (often a previously built `./chess`) is still running. Stop it and run `make dev` again.

## Production build

```sh
make build      # builds web, embeds it, writes ./chess
DATABASE_URL=postgres://user:pass@host:5432/chess ./chess   # serves everything on :8080 (use -addr :3000 to change)
```

`make build-linux` also writes `chess-linux-amd64` for a droplet.

## Droplet (later)

1. Install Postgres and make a user and database for the app:

   ```sh
   sudo apt install postgresql
   sudo -u postgres createuser --pwprompt chess   # pick a password
   sudo -u postgres createdb --owner chess chess
   ```

2. Copy `chess-linux-amd64` to the droplet, for example `/opt/chess/chess`. It applies migrations itself when it starts. To apply them on their own, for example before switching over, run it with `-migrate`.
3. Run it with systemd:

   ```ini
   # /etc/systemd/system/chess.service
   [Unit]
   Description=chess
   After=network.target postgresql.service

   [Service]
   Environment=DATABASE_URL=postgres://chess:YOUR_PASSWORD@localhost:5432/chess
   ExecStart=/opt/chess/chess -addr 127.0.0.1:8080
   Restart=always
   DynamicUser=yes

   [Install]
   WantedBy=multi-user.target
   ```

4. Put Caddy in front for HTTPS and compression:

   ```
   chess.example.com {
       encode zstd gzip
       reverse_proxy 127.0.0.1:8080
   }
   ```

Accounts and games live in Postgres, which survives redeploys; back it up with `sudo -u postgres pg_dump chess > chess-backup.sql`. Idle matches leave memory after 2 hours and load again from the database when someone opens them.

## Protocol

- `POST /api/matches?player=<browser id>` with an optional body `{"fog":true,"kingCapture":true,"setup":"battleship","jumpKing":true,"backPawns":true,"hill":true,"unrated":false,"time":{...}}` (`setup` is `""`, `"backRow"` or `"battleship"`) returns `{"id":"K7QF2M"}`. The creator gets white.
- `GET /api/matches/{id}` returns 204 if the match exists, 404 if it doesn't
- `GET /ws/{id}?player=<browser id>`: the server sends `{"type":"state", ...}` to each viewer after every change, built for that viewer:
  - `board`: 64 characters from a1 to h8. FEN letters for pieces (uppercase is white), `.` for empty, and `?` for squares the viewer can't see (reserved for fog of war).
  - `settings`, `turn`, `ply`, `last` (`{from,to}`), `check` (king square), `history` (SAN, `?` for moves hidden by fog).
  - `players`: `{"w":{"taken":true,"online":true,"name":"Uriel"},"b":{...}}`, where `name` is the username of a signed-in player and absent for a guest.
  - `captured`: `{"w":"qnp","b":"P"}`, the pieces each side has taken as FEN letters, most valuable first. Sent to everyone, since players know this even under fog.
  - `legal`: UCI moves the viewer may play now, sent only to the player whose turn it is.
  - `status`: `waiting`, `setup`, `playing` or `over`. During `setup` the board is empty and `setup` is `{"ready":{"w":true,"b":false},"mine":"<your 64 squares>","theirs":"<theirs, only without fog once they're ready>"}`. A finished game won on the first move has `"sniper":true`.
  - `spectators`: how many people are watching, not counting the players. Spectators also get `watching` (`w` or `b`, the side they see the game as) and, in a fog game, `behind` (how many moves they aren't shown yet).
- The client sends `{"type":"move","uci":"e7e8q"}`, `{"type":"draw"}`, `{"type":"resign"}`, `{"type":"rematch"}`, or `{"type":"ping","t":<ms>}`, which the server answers with `{"type":"pong","t":<same>}`. The draw and rematch messages toggle the offer.
- During setup, players send `{"type":"setup","board":"<64 squares, a1 to h8, only their own pieces>"}` to say they're ready, or an empty `board` to take it back and change it.
- Spectators send `{"type":"watch","side":"b"}` to see the game as the other side. Every spectator starts as White.
- Players send `{"type":"chat","text":"gg"}`. The server sends players `{"type":"chat","reset":true,"lines":[...]}` with the whole chat when they connect and when a rematch clears it, then `{"type":"chat","lines":[{"id":3,"from":"w","text":"gg","at":<unix ms>}]}` for each new message.
- A slow connection skips straight to the newest state instead of queueing old ones. Chat messages and pongs are never skipped.

## Measuring sync speed

The match page shows `ping` (round trip to the server) and how long the server took to confirm your last move. Run the server with `./chess -v`, or `cd server && go run . -v`, to log every action with how long it took to handle (about 0.1 ms per move).

## Credits

Piece set "cburnett" by Colin M.L. Burnett, CC BY-SA 3.0.
