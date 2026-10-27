SHELL := bash

# Local dev: Go server on :8080 and Vite on :5173 (open http://localhost:5173).
# Refuses to start if something (like an older ./chess) already holds :8080, since
# the new web app would otherwise talk to that old server. If either half exits,
# both stop.
dev:
	@if curl -s -o /dev/null --max-time 1 http://localhost:8080/; then \
		echo "Port 8080 is already in use, probably by an older ./chess or make dev. Stop it first."; exit 1; fi
	@cd web && npm install --no-audit --no-fund --loglevel=error
	@trap 'kill 0' EXIT; (cd server && go run .) & (cd web && npm run dev -- --host) & wait -n

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

.PHONY: dev build build-linux run
