SHELL := bash
.DEFAULT_GOAL := help

# ---- Project ----------------------------------------------------------------
# fow-chess: Go API (server/) + Vite web app (web/). Ports from PORTS.md in
# streamlinedesign/template; .env overrides.
NAME     ?= chess
PORT     ?= 4100
WEB_PORT ?= 4101
DROPLET  ?=
DB       ?= chess
# -----------------------------------------------------------------------------

-include .env
export

VERSION := $(shell cat VERSION 2>/dev/null || echo 0.0.0)
LDFLAGS := -s -w -X main.version=$(VERSION)

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'

# The server is compiled before Vite starts, so the page never comes up before
# the API it talks to. If either half exits, both stop.
dev: ## Go server on PORT, Vite on WEB_PORT (open http://localhost:4101)
	@if curl -s -o /dev/null --max-time 1 http://127.0.0.1:$(PORT)/; then \
		echo "Port $(PORT) is already in use, probably by an older ./chess or make dev. Stop it first."; exit 1; fi
	@cd web && npm install --no-audit --no-fund --loglevel=error
	@echo "Building server..."
	@cd server && go build -o ../.dev-server .
	@echo "On your phone: http://$$(hostname -I 2>/dev/null | cut -d' ' -f1 || ipconfig getifaddr en0):$(WEB_PORT)"
	@trap 'kill 0' EXIT; ./.dev-server & srv=$$!; \
		until curl -s -o /dev/null --max-time 1 http://127.0.0.1:$(PORT)/; do kill -0 $$srv 2>/dev/null || exit 1; sleep 0.1; done; \
		(cd web && npm run dev -- --host) & wait -n

test: ## Server tests
	cd server && go test ./...

build: ## One self-contained binary at ./chess
	cd web && npm ci && npm run build
	rm -rf server/dist && cp -r web/dist server/dist
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o ../$(NAME) .

lint: ## gofmt, go vet, eslint
	gofmt -l -w $$(git ls-files '*.go')
	cd server && go vet ./...
	cd web && npm run lint --if-present

# The server and tests use DATABASE_URL, or else a database called chess in the
# Postgres installed on this machine, which the server creates on first start.
migrate: ## Create the database if needed and apply migrations
	cd server && go run . -migrate

db-reset: ## Drop the local chess database and recreate it
	@if [ -n "$$DATABASE_URL" ]; then echo "DATABASE_URL is set; db-reset only resets the local $(DB) database."; exit 1; fi
	dropdb --if-exists $(DB)
	$(MAKE) migrate

db-seed: ## Load demo accounts from seed/ (password: demo)
	@for f in seed/*.sql; do echo "$$f"; psql -v ON_ERROR_STOP=1 -q "$${DATABASE_URL:-postgres:///$(DB)}" -f "$$f"; done

bugs: ## Print the newest bug reports (N=50), from DATABASE_URL or the local database
	cd server && go run . -bugs $(or $(N),50)

psql: ## A psql shell on the database
	psql "$${DATABASE_URL:-postgres:///$(DB)}"

deploy: ## Build for linux/amd64 and ship to DROPLET
ifeq ($(DROPLET),)
	@echo "Set DROPLET=user@host in .env"; exit 1
endif
	cd web && npm ci && npm run build
	rm -rf server/dist && cp -r web/dist server/dist
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o ../$(NAME)-linux-amd64 .
	scp $(NAME)-linux-amd64 $(DROPLET):/tmp/$(NAME)
	ssh $(DROPLET) 'sudo install -m 755 /tmp/$(NAME) /usr/local/bin/$(NAME) && sudo systemctl restart $(NAME) && systemctl is-active $(NAME)'

release: ## Tag VERSION, move Unreleased in CHANGELOG.md, push the tag
	@test "$$(git branch --show-current)" = main || { echo "release from main only"; exit 1; }
	@git diff --quiet || { echo "commit first"; exit 1; }
	@grep -q "## \[Unreleased\]" CHANGELOG.md || { echo "CHANGELOG.md needs an Unreleased section"; exit 1; }
	sed -i.bak "s/## \[Unreleased\]/## [Unreleased]\n\n## [$(VERSION)] - $$(date +%Y-%m-%d)/" CHANGELOG.md && rm CHANGELOG.md.bak
	git commit -am "Release $(VERSION)"
	git tag -a v$(VERSION) -m "v$(VERSION)"
	git push && git push --tags

configure-repo: ## One-time GitHub settings: auto-merge, Actions may open PRs, main protected, hold label
	@repo=$$(gh repo view --json nameWithOwner --jq .nameWithOwner); \
	gh api -X PATCH repos/$$repo -F allow_auto_merge=true -F delete_branch_on_merge=true -F allow_squash_merge=true >/dev/null; \
	gh api -X PUT repos/$$repo/actions/permissions/workflow -F default_workflow_permissions=write -F can_approve_pull_request_reviews=true >/dev/null; \
	gh api -X PUT repos/$$repo/branches/main/protection --input - >/dev/null <<< '{"required_status_checks":{"strict":false,"contexts":["ci"]},"enforce_admins":false,"required_pull_request_reviews":null,"restrictions":null,"allow_force_pushes":false,"allow_deletions":false}'; \
	gh label create hold --color B60205 --description "Do not auto-merge" --force >/dev/null; \
	echo "$$repo configured"

clean: ## Remove build output and the dev server binary
	rm -rf web/dist server/dist/* .dev-server $(NAME) $(NAME)-linux-amd64

.PHONY: help dev test build lint migrate db-reset db-seed bugs psql deploy release configure-repo clean
