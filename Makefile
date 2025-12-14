SHELL := /bin/sh
GO ?= go
BINARY := githut
CMD := ./cmd/githut
DIST := dist

.PHONY: dev build

dev:
	@command -v air >/dev/null 2>&1 && air -c .air.toml || $(GO) run $(CMD) serve

build:
	@mkdir -p $(DIST)
	$(GO) build -trimpath -o $(DIST)/$(BINARY) $(CMD)
