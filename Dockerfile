# syntax=docker/dockerfile:1.6
FROM golang:1.25-alpine AS builder
WORKDIR /src
ARG CMD_PATH
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build sh -c '\
  set -eux; \
  CMD="${CMD_PATH:-}"; \
  if [ -z "$CMD" ]; then \
    mains="$(go list -f '\''{{if eq .Name "main"}}{{.Dir}}{{end}}'\'' ./... | grep . || true)"; \
    count="$(printf "%s\n" "$mains" | sed "/^$/d" | wc -l)"; \
    if [ "$count" -eq 0 ]; then echo "ERROR: No main packages found in module"; exit 1; fi; \
    if [ "$count" -gt 1 ]; then echo "ERROR: Multiple main packages found:"; printf "%s\n" "$mains"; exit 1; fi; \
    CMD="$mains"; \
  fi; \
  test -e "$CMD" || (echo "ERROR: CMD_PATH '\''$CMD'\'' not found" && exit 1); \
  CGO_ENABLED=0 go build -o /out/githut "$CMD" \
'

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/githut /usr/local/bin/githut
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s CMD wget -qO- http://localhost:8080/readyz || exit 1
ENTRYPOINT ["/usr/local/bin/githut"]
CMD ["serve"]
