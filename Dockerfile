# syntax=docker/dockerfile:1.6
FROM golang:1.25-alpine AS builder
WORKDIR /src
ARG CMD_PATH=./cmd/githut
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN test -e "${CMD_PATH}" || (echo "ERROR: CMD_PATH '${CMD_PATH}' not found. Ensure your repo contains the main package at this path." && exit 1)
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /out/githut ${CMD_PATH}

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/githut /usr/local/bin/githut
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s CMD wget -qO- http://localhost:8080/readyz || exit 1
ENTRYPOINT ["/usr/local/bin/githut"]
CMD ["serve"]
