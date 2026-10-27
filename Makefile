SHELL := bash

# Local dev: Postgres in Docker, Go server on :8080 and Vite on :5173 (open http://localhost:5173).
# Refuses to start if something (like an older ./chess) already holds :8080, since
# the new web app would otherwise talk to that old server. The server is compiled
# before Vite starts, so the page never comes up before the API it talks to (a
# cold Go build takes several seconds). If either half exits, both stop.
dev: db
	@if curl -s -o /dev/null --max-time 1 http://localhost:8080/; then \
		echo "Port 8080 is already in use, probably by an older ./chess or make dev. Stop it first."; exit 1; fi
	@cd web && npm install --no-audit --no-fund --loglevel=error
	@echo "Building server..."
	@cd server && go build -o ../.dev-server .
	@trap 'kill 0' EXIT; ./.dev-server & srv=$$!; \
		until curl -s -o /dev/null --max-time 1 http://localhost:8080/; do kill -0 $$srv 2>/dev/null || exit 1; sleep 0.1; done; \
		(cd web && npm run dev -- --host) & wait -n

# Postgres for dev and tests, from docker-compose.yml. Skipped when DATABASE_URL
# is set, since that means you're bringing your own.
db:
	@if [ -n "$$DATABASE_URL" ]; then echo "Using DATABASE_URL, not starting Docker Postgres."; exit 0; fi; \
	if ! docker info >/dev/null 2>&1; then echo "Docker isn't running. Start Docker Desktop (or set DATABASE_URL to your own Postgres)."; exit 1; fi; \
	docker compose up -d --wait db

db-stop:
	docker compose stop db

# Deletes every account and game in the local database.
db-reset:
	docker compose down -v db
	$(MAKE) db

# A psql shell on the local database.
psql: db
	docker compose exec db psql -U chess chess

test: db
	cd server && go test ./...

# Production: one self-contained binary at ./chess that serves the app on :8080.
build:
	cd web && npm ci && npm run build
	rm -rf server/dist && cp -r web/dist server/dist
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../chess .

# Same, cross-compiled for a typical Linux droplet.
build-linux:
	$(MAKE) build
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../chess-linux-amd64 .

run: build
	./chess

.PHONY: dev db db-stop db-reset psql test build build-linux run
