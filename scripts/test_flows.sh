#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

log() { printf "[%s] %s\n" "$(date +%H:%M:%S)" "$*"; }

BASE_URL="${GITHUT_BASE_URL:-http://localhost${GITHUT_HTTP_ADDR:-:8080}}"

require_env() {
  if [[ -z "${GITHUT_POSTGRES_DSN:-}" ]]; then
    log "ERROR: GITHUT_POSTGRES_DSN is not set. Export it to run DB-backed flows."
    log "Example: export GITHUT_POSTGRES_DSN='postgres://user:pass@localhost:5432/githut?sslmode=disable'"
    exit 1
  fi
}

cleanup() {
  if [[ -n "${SERVE_PID:-}" ]]; then
    log "Stopping server PID ${SERVE_PID}"
    kill "${SERVE_PID}" || true
    wait "${SERVE_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

start_server() {
  log "Starting server..."
  GITHUT_HTTP_ADDR="${GITHUT_HTTP_ADDR:-:8080}" \
  GITHUT_POSTGRES_DSN="${GITHUT_POSTGRES_DSN}" \
  nohup bash -c 'go run ./cmd/githut serve' >/tmp/githut-serve.log 2>&1 &
  SERVE_PID=$!
  log "Server PID: ${SERVE_PID}"
  for i in {1..50}; do
    if curl -sSf "${BASE_URL}/readyz" >/dev/null; then
      log "Server ready"
      return 0
    fi
    sleep 0.2
  done
  log "ERROR: Server not ready; logs:"
  tail -n +1 /tmp/githut-serve.log || true
  exit 1
}

git_with_header() {
  local header="$1"; shift
  git -c "http.extraHeader=${header}" "$@"
}

main() {
  require_env

  log "Running migrations"
  go run ./cmd/githut db migrate up

  log "Creating users"
  go run ./cmd/githut users create --username alice --email alice@example.com --role developer --password 'secret123'
  go run ./cmd/githut users create --username bob --email bob@example.com --role developer --password 'hunter2'

  log "Creating repositories"
  go run ./cmd/githut repos create --owner alice --name demo --visibility public
  go run ./cmd/githut repos create --owner alice --name private-repo --visibility private
  log "Adding member bob to private-repo"
  go run ./cmd/githut repos members add --owner alice --name private-repo --username bob --role developer

  log "Issuing token for alice"
  ALICE_TOKEN="$(go run ./cmd/githut users token create --username alice --name testcli | tail -n 1)"
  if [[ -z "${ALICE_TOKEN}" ]]; then
    log "ERROR: Failed to create token for alice"
    exit 1
  fi

  start_server

  WORKDIR="$(mktemp -d -t githut-test-XXXXXX)"
  log "Working directory: ${WORKDIR}"
  pushd "${WORKDIR}" >/dev/null

  log "Cloning public repo over HTTP"
  git clone "${BASE_URL}/git/alice/demo" demo
  pushd demo >/dev/null
  log "Commit and push using Basic auth"
  git config user.name "Alice"
  git config user.email "alice@example.com"
  echo "hello $(date +%s)" > README.md
  git add README.md
  git commit -m "Add README"
  B64="$(printf "%s" "alice:secret123" | base64 -w 0)"
  git_with_header "Authorization: Basic ${B64}" push origin HEAD:refs/heads/main || git_with_header "Authorization: Basic ${B64}" push origin HEAD
  popd >/dev/null

  log "Cloning private repo over HTTP with Bearer token"
  git_with_header "Authorization: Bearer ${ALICE_TOKEN}" clone "${BASE_URL}/git/alice/private-repo" private-repo
  pushd private-repo >/dev/null
  log "Commit and push to private with Bearer token"
  git config user.name "Alice"
  git config user.email "alice@example.com"
  echo "secret $(date +%s)" > PRIVATE.md
  git add PRIVATE.md
  git commit -m "Add private content"
  git_with_header "Authorization: Bearer ${ALICE_TOKEN}" push origin HEAD:refs/heads/main || git_with_header "Authorization: Bearer ${ALICE_TOKEN}" push origin HEAD
  popd >/dev/null

  log "Fetch metrics"
  curl -sSf "${BASE_URL}/metrics" | head -n 20

  popd >/dev/null
  log "Show audits for demo"
  go run ./cmd/manage admin audits --owner alice --name demo
  log "Flow completed successfully"
}

main "$@"
