CLI_MAIN_PKG := ./cmd/cli/main.go
SERVER_MAIN_PKG := ./cmd/server/main.go
CLI_BINARY_NAME := stasis-cli
SERVER_BINARY_NAME := stasis-server


## caddy-dev: run the caddy server
.PHONY: caddy-dev
caddy-dev:
	@command -v caddy >/dev/null 2>&1 || (echo "caddy is not installed" && exit 1)
	@caddy run --config deploy/Caddyfile

## web: run the web development server
.PHONY: web
web:
	@source configs/.env && cd web && bun run dev

## dev: run the frontend & backend in development environment
.PHONY: dev
dev:
	@echo "Running in development environment..."
	@make -j2 watch-server watch-cli

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
	@CONFIG_PATH=configs/config.yaml go build -o bin/$(CLI_BINARY_NAME) $(CLI_MAIN_PKG)

## build-server: build server for production
.PHONY: build-server
build-server:
	@echo "Building server..."
	@CONFIG_PATH=configs/config.yaml go build -o bin/$(SERVER_BINARY_NAME) $(SERVER_MAIN_PKG)


## watch-server: run the server application with reloading on file changes
.PHONY: watch-server
watch-server:
	@if command -v air > /dev/null; then \
		    air --build.cmd "make build-server" --build.bin "bin/stasis-server"; \
		    echo "Watching...";\
		else \
		    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
		        go install github.com/air-verse/air@latest; \
		        air --build.cmd "make build-server" --build.bin "bin/stasis-server"; \
		        echo "Watching...";\
		    else \
		        echo "You chose not to install air. Exiting..."; \
		        exit 1; \
		    fi; \
		fi

## watch-cli: run the cli with reloading on file changes
.PHONY: watch-cli
watch-cli:
	@if command -v air > /dev/null; then \
		    air --build.cmd "make build-cli" --build.bin "bin/stasis-cli"; \
		    echo "Watching...";\
		else \
		    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
		        go install github.com/air-verse/air@latest; \
		        air --build.cmd "make build-cli" --build.bin "bin/stasis-cli"; \
		        echo "Watching...";\
		    else \
		        echo "You chose not to install air. Exiting..."; \
		        exit 1; \
		    fi; \
		fi

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
