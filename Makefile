SHELL := /bin/sh
GO ?= go
BINARY := githut
CMD := ./cmd/githut
DIST := dist
GITHUT_ADDR := localhost:8080
FRONTEND_ADDR := localhost:5173

.PHONY: dev build

caddy-dev:
	@command -v caddy >/dev/null 2>&1 || (echo "caddy is not installed" && exit 1)
	@caddy run --config deploy/Caddyfile

server:
	@command -v air >/dev/null 2>&1 && air -c .air.toml || $(GO) run $(CMD) --config githut.yaml serve

web:
	@cd web
	@bun run dev

build:
	@mkdir -p $(DIST)
	$(GO) build -trimpath -o $(DIST)/$(BINARY) $(CMD)
