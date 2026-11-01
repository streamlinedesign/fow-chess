# House rules for every Streamline project

These rules are the same in every `streamlinedesign` repo. Follow them before
anything project-specific below the line.

## Workflow

- Start from `main`. Work on a branch named `claude/<short-topic>` (the
  default for Claude threads) or `feat/<topic>` for a person.
- Before pushing: `make test` and `make build` must pass. No exceptions.
- Every push to a `claude/*` or `feat/*` branch opens a pull request by
  itself (`.github/workflows/auto-pr.yml`). CI runs `make test` and
  `make build`. When CI is green the PR merges and the branch is deleted.
  Add the label `hold` to a PR to stop it merging on its own.
- Never push straight to `main`. Never force-push a shared branch.
- One PR is one thing. Keep the description to what changed and why.
- A PR that changes behaviour bumps `VERSION` and adds a line to
  `CHANGELOG.md` under `Unreleased`. `make release` turns Unreleased into a
  dated, tagged version.

## The Makefile is the only entry point

Every repo has the same targets. Never tell a person to run a raw
`go run`, `npm`, `cmake` or `docker` command when a target exists; add the
target instead.

| Target | Does |
|---|---|
| `make dev` | Runs the project locally on its port(s) from `PORTS.md`, reloading on change. |
| `make test` | All tests. Must pass before any push. |
| `make build` | Production build into `./dist` or one binary at `./<name>`. |
| `make lint` | Formatters and linters; fixes what it can. |
| `make db-reset` | Drops and recreates the local database with migrations. |
| `make db-seed` | Loads demo data from `seed/` so the app is usable at once. |
| `make deploy` | Builds for Linux and ships to the droplet named in `.env`. |
| `make release` | Tags `VERSION`, moves Unreleased in the changelog, pushes the tag. |
| `make clean` | Removes build output and local databases. |

A target that doesn't apply (no database, nothing to deploy) stays in the
Makefile and prints `not used in this project`, so `make <anything>` never
fails with "no rule".

## Ports

Each project owns a block of ten ports, listed in `PORTS.md` in the
template repo. Nothing runs on 8080, 3000, 5173 or any tool's default.
`make dev` reads `PORT` (and `WEB_PORT` where there is a separate dev
server) from `.env`; `.env.example` holds the project's numbers. In
production the service listens on `127.0.0.1:$PORT` behind Caddy.

## Secrets and config

- `.env` is local and git-ignored. `.env.example` is committed and lists
  every variable with a safe placeholder. Keep the two in step.
- On a server the same variables live in `/etc/<name>/env`, read by the
  systemd unit's `EnvironmentFile`. Never in the unit file, never in Caddy.
- API tokens come from the owner's password manager, nowhere else. Never
  commit one, never paste one into a PR, log or chat. If one leaks, say so
  immediately so it can be rotated.
- CI secrets go in the repo's Actions secrets.

## Databases and demo data

Where a project has a database (Postgres by default, SQLite for small
single-binary tools):

- Migrations live in `migrations/` and run on start and on `make db-reset`.
- `seed/` holds demo data that makes the app presentable in one command:
  a few realistic rows per table, plus one account `demo` / `demo`.
  `make db-seed` loads it; `make db-reset` gives a clean slate. Keep the
  seed updated when the schema changes, or `make test` should fail.

## Todo

`TODO.md` at the repo root is the project's working list: a short `Now`
section, a `Next` section, and `Later`. Move an item to `Done` with the
version it shipped in. When starting work, read it; when finishing, update
it. The project dashboard reads this file.

## Style

Streamline's visual language is **Blueprint Paper**
(`streamlinedesign/blueprint-paper`): navy paper, pale ink, one monospace
face (IBM Plex Mono), line work that shows state, cards that invert on
hover. For anything with a UI:

- Use the kit: `<link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/streamlinedesign/blueprint-paper@<version>/dist/blueprint-paper.min.css">`,
  pinned to a tagged version, body class `bp-sheet`. Use its tokens
  (`--bp-paper`, `--bp-pen`, the ink ramp, `.bp-corners`) rather than new
  colours.
- Don't invent a second palette or a second typeface in a project. If the
  kit lacks something, add it to blueprint-paper in its own PR and bump
  the kit's version.
- Pages must work at 360px wide, in dark mode, and with keyboard only.
- Copy is sentence case, short, no exclamation marks.

## Deploy

Production is a Debian droplet: one Caddy in front, each project a
systemd service as its own user, listening on `127.0.0.1:$PORT`, data in
`/var/lib/<name>`, config in `/etc/<name>/env`. `deploy/` in the repo holds
the unit file and the Caddy site block; `make deploy` builds for
`linux/amd64`, copies the binary and restarts the service. A project that
is a static site deploys to GitHub Pages instead.

## Writing

READMEs and docs are plain, concrete and short. Say what a thing is, how to
run it, and what is still open. No marketing language.

---

## This project

<!-- Project-specific notes go here: what it is, its ports, what's unusual. -->

**fow-chess** is Eat the King: two-player online chess with fog of war.
`web/` is React 19 + Vite + Tailwind v4 and knows no chess rules; `server/`
is Go + Postgres and holds every live match in memory; `server/engine/` is
the dependency-free rules engine (also copied into `chess-bots`, keep them
in step until it becomes a module).

- Ports: server `4100`, Vite `4101`. Open http://localhost:4101 in dev.
- `make db-reset && make db-seed` gives a clean database with accounts
  `demo`, `alice`, `bob`, `carol`, `dave` (password `demo`).
- Schema changes are a new numbered file in `server/migrations/`, never an
  edit to one that shipped. Update `seed/` in the same PR.
- The server forces fog of war and Eat the King on every match
  (`normalize` in `server/view.go`); don't add options that bypass it.
- `.env.example`: `PORT`, `WEB_PORT`, `DATABASE_URL`, `DROPLET`.
