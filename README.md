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
- **Modes** (`/mods` only) decide where the pieces start: **Standard** (the usual position), **Back row** (each player arranges their back row; pawns stay put) or **Ambush** (each player places all sixteen pieces anywhere in their own four rows). Both players set up at the same time, and the clock doesn't run until both are ready. You never see the other setup. Eat the King on the home page always starts from the usual position (`normalize` in `server/view.go` drops the layout). Castling is allowed only for a king and rook that start on their usual squares. Setups are checked in `server/setup.go`, and a game saves the position it started from (`start_fen`) so replays and restarts work.
- **Ranked**: every game of the one game is ranked when both players are signed in. There's no switch for it: `normalize` drops any `unrated` a request sends. Mods games and bot games are never ranked.
- **Find match** (`POST /api/queue`, `server/queue.go`): a signed-in player waits for the player nearest their rating who wants the same clock and mode. The rating window starts at 100 points and widens by 20 a second, so after half a minute anyone wanting the same game will do. The page polls `GET /api/queue` every 3 seconds, which is also what keeps a player in the queue; the pair is told over live events (`match`). Colours are random.
- **Bots** (`server/bots`, `server/bot.go`): four computer opponents, weakest first: Rex inutilis (a random move), Vercingetorix (2 plies), Ramesses II (3 plies) and Solomon (4 plies, and it values seeing the board). Ramesses II and Solomon also count the ways into their own king from squares they can't see (`exposure` in `server/bots/fog.go`), since in the simulations about four kings in five were taken by a piece their owner never saw; with it, Ramesses II scores about 0.59 against itself without it over 2,000 fog games (`BOTS_MATCH=2000 go test ./bots -run GuardedBeatsPlain -v`). Remembering where enemy pieces were last seen and closing in on the king were tried and didn't help, so they're not in. A bot plays from its own view: under fog the pieces it can't see are left off the board it searches, so it never cheats; its candidate moves are the real legal ones, so it never plays an illegal move. A bot holds a seat with the key `bot:<level>` and `bot` in the settings, so a bot game survives a restart; the match pokes it whenever something changes and it answers after a short pause (`botDelay`), sets up at random in Back row and Battleship (in Battleship always with its king guarded, `bots.GuardedKing`: a random setup lost on the first move 42% of the time in the simulations), and accepts rematches. You get a random colour. Bot games are never ranked. Find match suggests a bot after 20 seconds with nobody around.
- **Sync Move** (mods, two or four players): both sides send a move each round without seeing the other's, then both happen at once. Moving pieces leave first and captures come last, so a piece that moves away dodges; two pieces landing on the same square knock each other out; if both kings fall it's a draw. A sent move is locked in, both clocks run until each side has sent, and there's no en passant or check (always Eat the King). The rules are `PlayRound` in `server/engine/sync.go`; a round is saved as one token, like `s:e2e4,e7e5`, and adds two plies to the history (White's, then Black's, an empty one for a side with no move).
- **War Mode** (home page, ranked): the second official game next to Eat the King, one fixed game, the cross-shaped board with Battleship and jester pawns, taking turns clockwise (the extended board and Sync Move are mods, not part of it), in the fog of war with centre vision (the four middle squares), 30 minutes each plus 10 seconds a move (`warMode` and `rankedFour` in `server/four_match.go`). Signed-in players wait in a queue of their own (`server/four_queue.go`, `/api/four/queue`), and the four closest four-player ratings are seated once their spread is within the window of whoever has waited longest (the same window as one-against-one matchmaking), in a random order. Everyone has two minutes to place their pieces, then gets a bot's arrangement. The game moves a separate Glicko-2 rating (`four_rating` and friends on `users`, `server/four_rating.go`): each player scores a win against everyone who finished below them, a draw against anyone level and a loss against everyone above (`four.Game.Places`: the winner, then the last to go out). The changes are saved on the game (`four_games.rating_diff`), and the leaderboard and profile show the War Mode rating apart (the API still calls it `four`: `/api/leaderboard?four=1`, `fourRating`, `fourRank`).
- **Four players** (`/mods`, then Four players): a cross-shaped board, 14 by 14 with the corners cut off (a full 8 by 8 in the middle and three rows on each side). Red, Blue, Green and Violet move clockwise, every player sees the usual setup from their own side, and there's no check: eating a king knocks that player out and takes their pieces off the board, as do resigning and running out of time. The last king standing wins. Fog of war, King of the Hill, Prince, backward and jester pawns, centre vision (the four middle squares) and every clock work as in two-player games; checkmate doesn't. **Back row** and **Ambush** work as in two-player games, in each player's three rows (`server/four/setup.go`; the arrangements are saved in `four_games.start`). **Sync Move** (four players only) has everyone send a move each round, then plays them all at once: every moving piece leaves its square first, a piece landing on a square another player's piece stayed on takes it, and two or more landing on the same square are all taken, with whatever stood there (`PlayRound` in `server/four/sync.go`). A round is saved as one token, like `s:e2e4,-,g13g11,m8k8`. En passant is off in Sync Move. Pawns promote on the eighth rank from their side, just past the middle; castling and en passant work. A seated player can fill the empty seats with bots (a greedy one-move bot that plays from its own view). The rules are `server/four` (its own engine, apart from `server/engine`); matches are `server/four_match.go`, saved in `four_games` and served at `/f/<id>` over `/ws/four/<id>`. Never ranked. **Extended board** (Board: Extended): 16 by 16, four rows each, and the four by four corners are neutral squares anyone may cross (`four.Shape`). Centre vision comes from your own two of the middle four, the pair just past the middle on the far side; pawns promote past the middle (the ninth rank from their side on), or on the side walls where the players to their left and right sit; Battleship setups are 32 squares; there's no King of the Hill. Square names stay the same on both boards (the cross runs d1 to k14, the extended board a1 to p16), and the state message sends width by width squares.
- **Mods** (`/mods`): the game with every dial exposed: fog off, checkmate rules, King of the Hill, Prince, backward pawns, and the extra clocks including No clock (`modTimeControls`). A mods game carries `mods: true`, keeps its options through `normalize`, and is never ranked.

A king that starts in danger after setup is left as it is. A win on the winner's very first move is a **Sniper Win**: it counts, but moves both ratings only a quarter as far as a whole game (`sniperWeight` in `server/setup.go`).

The older options (checkmate rules, no fog, King of the Hill, Prince, backward pawns: `Rules` in `server/engine`) live on as mods, and games saved with them before the site became one game keep working and replaying. They're described with the `legacy` labels in the dictionaries.

The rules live in `server/engine` (`Rules`, `Position.Vision`), and what each viewer receives is decided in `server/view.go`.

## Accounts and saved games

Players can sign up with just a username and password (no email). Passwords are hashed with bcrypt, and sessions are a random token in an http-only cookie, with only a hash of it stored. Guests can still play without an account.

Everything lives in Postgres. The server connects to `DATABASE_URL` (or `-db <url>`), and defaults to a database called `chess` in the Postgres installed on your machine. On startup it creates that database if it's missing, then creates and updates the tables from the numbered SQL files in `server/migrations/`: to change the schema, add a new file rather than editing one that has shipped.

- Every game is saved as it's played, so a server restart no longer ends games: reopen the link and play on.
- Games with at least one signed-in player are kept forever. Games between two guests are deleted after 30 days.
- A guest who signs in mid-game keeps their seat.
- `/u/<username>` shows a player's rating, current games and finished games. Finished games can be replayed move by move at `/g/<match id>/<round>`, with the fog lifted. Games still being played can't be replayed, since that would reveal the board.
- The home page is one big board, full width like the header, its cells split by hairlines. On top, a **demo board**: a game against Ramesses II that's playable the moment the page loads, as White or Black, with no clock. It's a match with `demo` in the settings (`POST /api/matches` with `{"demo": true, "side": "w"}`), kept only in memory: never saved, listed on Watch, ranked, or shown as playing, and dropped 10 minutes after you leave. A toggle above the bots switches it to **War Mode** against three copies of the picked bot (`POST /api/four` with `{"demo": true, "bot": "solomon"}`, `demoFour`): the same rules as ranked War Mode with no clock, opening on your Battleship setup while the bots have already placed theirs; the War Mode card's "Try it against the bots" switches to it too. For now every four-player bot but Rex inutilis (random moves) plays the same one-move bot, `four.BotMove`. Beside it the play menu: Find an opponent (Short: Blitz and Rapid; Long: Tournament, Daily, Weekly; the mode; one Play button, ranked), then a quieter block to play a friend with a code, then Mods. Below: your games in progress, the next arena, friends online, the bots, a few live games and the Top 100 (`GET /api/leaderboard`, which also returns your own rank when you're signed in). Guests see a prompt to sign in instead of friends. The rules and the Opera Game replay are at `/rules`.

## Clocks and rating

The creator picks a time control from fixed presets (`timeControls` in `server/view.go`, `TIME_GROUPS` in `web/src/settings.ts`, which must match): Blitz 5+0 and 3+2, Rapid 10+0, Tournament 90+30, Daily (24 hours per move, reset after every move) or Weekly (7 days per move). The server refuses anything else for a new game; games saved with an earlier preset, including No clock, keep their clock and still play and replay.

- Clocks run on the server. White's clock starts once Black sits down. The increment is added after each move.
- When a clock runs out, that side loses on time, unless the opponent has only a king, or a king and one minor piece, which is a draw. A timer ends the game even with nobody connected, and every 30 seconds the server also checks the database for games whose deadline passed while it wasn't holding them (after a restart, for example).
- The browser counts down locally from the last state the server sent, so the clock stays smooth between messages.
- A game is **ranked** (rated) when both players are signed in, it isn't a mods game, isn't a bot game, and has a clock (every preset on offer has one). Ratings use Glicko-2, starting at 1500 with one rating for every mode, and each game is its own rating period. The change is applied once, in a transaction, when the finished game is saved, and shown in the game-over dialog and on the profile.

## Tournaments

An arena every evening, Zagreb time (`arenaKinds` in `server/tournament.go`): Rapid 10+0 at 19:00 for 45 minutes and Blitz 5+0 at 20:00 for 30 minutes. The server keeps the next two days on the calendar (`tournaments` table) and ticks every 3 seconds.

- Signed-in players **join** before the start or while it runs (`POST /api/tournaments/{id}/join`). At the start it needs at least 4 players (`minArenaPlayers`), or it's called off and everyone who joined hears in the bell.
- **Pairing is automatic:** players who are in, not sitting out, not playing and online (a tab open) are paired nearest in points first, avoiding an immediate rematch when there's someone else. The game is a normal ranked match with `tournament` in its settings; both players get a live `match` event and the app opens the game wherever they are.
- **Scoring:** 2 for a win (a Sniper Win is a win), 1 for a draw, recorded once per game in `tournament_games` and `tournament_players` through the match's `over` hook. A game that ends after the arena has finished doesn't count. Leaving a running arena sits you out; your points stay.
- When it ends, everyone gets their place in the bell. A running arena survives a restart: it's rebuilt from the tables, games still in play included.
- `/tournaments` shows what's coming, the running arena with its table (polled every 5 seconds), recent results and the rules; the home page shows the next arena.

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

Then open http://localhost:4101. On the first start the server creates a `chess` database and its tables, so there's nothing to set up by hand. It connects through Postgres's local socket as your own user, which is how Homebrew and Postgres.app install it. If your Postgres needs a user, password or another host, set `DATABASE_URL` instead, for example `export DATABASE_URL=postgres://chess:secret@localhost:5432/chess`.

- `make migrate` creates the database if needed and applies migrations, without starting the server.
- `make psql` opens a SQL shell on the database.
- `make db-reset` deletes every account and game in the local `chess` database and starts fresh.
- On Linux, where Postgres only knows the `postgres` user at first, give yourself a role once: `sudo -u postgres createuser --superuser $USER`.
- Accounts and games from the old SQLite `chess.db` aren't carried over.

To play against yourself, open a second browser or a private window, since each browser gets one seat. Vite is started with `--host`, so a phone on the same Wi-Fi can open `http://<your-computer-ip>:4101`.

## Tests

```sh
make test                          # runs every Go test
cd server && go test ./...         # includes perft to depth 4-5 on the standard test positions (~3s)
cd server && go test -short ./...  # skips the deep perft runs
```

Tests that touch the database each get their own Postgres schema, dropped afterwards, so they never touch your dev data. They use `CHESS_TEST_DATABASE_URL` if set, else the same database as the server, and are skipped with a note if Postgres isn't running.

If a page says **Server out of date** or `make dev` says port 4100 is in use, an older copy of the server (often a previously built `./chess`) is still running. Stop it and run `make dev` again.

## Production build

```sh
make build      # builds web, embeds it, writes ./chess
DATABASE_URL=postgres://user:pass@host:5432/chess ./chess   # serves everything on 127.0.0.1:4100 (PORT or -addr to change)
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
   ExecStart=/opt/chess/chess -addr 127.0.0.1:4100
   Restart=always
   DynamicUser=yes

   [Install]
   WantedBy=multi-user.target
   ```

4. Put Caddy in front for HTTPS and compression:

   ```
   chess.example.com {
       encode zstd gzip
       reverse_proxy 127.0.0.1:4100
   }
   ```

Accounts and games live in Postgres, which survives redeploys; back it up with `sudo -u postgres pg_dump chess > chess-backup.sql`. Idle matches leave memory after 2 hours and load again from the database when someone opens them.

## Protocol

- `POST /api/matches?player=<browser id>` with an optional body `{"fog":true,"kingCapture":true,"setup":"battleship","jumpKing":true,"hill":true,"unrated":false,"time":{...}}` (`setup` is `""`, `"backRow"` or `"battleship"`) returns `{"id":"K7QF2M"}`. The creator gets white.
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

The piece set was drawn for this project and lives in `web/src/pieces/`.
