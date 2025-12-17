CLI_MAIN_PKG := ./cmd/cli/main.go
SERVER_MAIN_PKG := ./cmd/server/main.go
CLI_BINARY_NAME := githut-cli
SERVER_BINARY_NAME := githut-server


## caddy-dev: run the caddy server
.PHONY: caddy-dev
caddy-dev:
	@command -v caddy >/dev/null 2>&1 || (echo "caddy is not installed" && exit 1)
	@caddy run --config deploy/Caddyfile

## web: run the web development server
.PHONY: web
web:
	@cd web
	@bun run dev

## dev: run the frontend & backend in development environment
.PHONY: dev
dev:
	@echo "Running in development environment..."
	@make -j2 watch dev

## update: updates the packages and tidy the modfile
.PHONY: update
update:
	@go get -u ./...
	@go mod tidy -v

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	@echo "Tidying up..."
	@go fmt ./...
	@go mod tidy -v

## build-cli: build cli for production
.PHONY: build-cli
build-cli:
	@echo "Building CLI..."
	@go build -o bin/$(CLI_BINARY_NAME) $(CLI_MAIN_PKG)

## build-server: build server for production
.PHONY: build-server
build-server:
	@echo "Building server..."
	@go build -o bin/$(SERVER_BINARY_NAME) $(SERVER_MAIN_PKG)

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
