# Implementation Plan

## Vision
Deliver a lightweight Git hosting service with SSH and Smart HTTP support, backed by PostgreSQL for identity and metadata, and S3-compatible object storage for repository data and LFS. Operate with clear separation of concerns and simple operability.

## Scope
- Protocols: SSH and Smart HTTP for `upload-pack` and `receive-pack`
- Manage CLI: users, repos, admin, stats
- Storage: S3 object storage and optional local cache
- DB: users, repos, permissions, SSH keys, audit logs
- Observability: health, logs, metrics
- Backups: S3 archives and retention policies

## Principles
- Keep components small and testable
- Prefer explicit configuration via environment variables
- Minimize external dependencies; standard library where possible
- Fail fast; return actionable errors

## Current Baseline
- HTTP health endpoint in `cmd/serve/main.go:1`
- Management CLI stubs in `cmd/manage/main.go:1`
- Configuration loader `internal/config/config.go:1`
- PostgreSQL connector `internal/database/db.go:1`
- Storage interface and S3 scaffold `internal/storage/*.go`
- Git server scaffold `internal/git/server.go:1`

## Phase Plan
### Phase 0: Scaffolding (Complete)
- Project structure and build pipelines
- Health endpoint and CLI scaffolding

### Phase 1: Smart HTTP
- Implement `git-upload-pack` and `git-receive-pack` HTTP endpoints
- Wire repo resolution, permission checks, and pack I/O
- Introduce local repo cache (`data/repos`) with S3 sync
- Add minimal authentication (Bearer token) for HTTP pushes
- Integration tests for fetch/push

### Phase 2: SSH
- Implement SSH server with `golang.org/x/crypto/ssh`
- Authorize via DB-backed SSH keys
- Execute `git-upload-pack` and `git-receive-pack` against repo paths
- Permission enforcement
- End-to-end tests with `ssh` client

### Phase 3: Manage CLI + DB
- Migrations for users, repos, permissions, keys, audit logs
- Implement CRUD operations in CLI backed by DB
- Add flags and validation
- Seed data and sample workflows

### Phase 4: LFS
- Implement LFS endpoints and storage mapping to S3
- Enforce permissions and auth
- Tests for pointer and object flows

### Phase 5: Observability & Audit
- Health, readiness, Prometheus metrics
- Structured logs
- Audit log writes for pushes, key changes, and admin actions

### Phase 6: Backup & Retention
- Periodic repository archive to S3
- Configurable retention and restore commands

## Work Breakdown
- HTTP protocol handlers: endpoints, request/response framing, sideband
- SSH daemon: key auth, exec, session management
- Repo storage: local cache synchronization and S3 mapping
- DB: migrations, queries, indices, permissions model
- CLI: commands, flags, validations, error reporting
- AuthN/AuthZ: tokens for HTTP, RBAC enforcement across protocols
- Observability: metrics, logs, healthz
- Backups: archive format, retention, restore tooling

## Deliverables & Acceptance
- Smart HTTP push/pull against a sample repo with auth and permissions
- SSH push/pull with authorized keys
- CLI managing users, repos, keys; accurate DB state
- LFS upload/download with storage in S3
- Audit logs present for critical actions
- Backups available and restorable

## Testing Strategy
- Unit tests for storage and DB abstractions
- Integration tests for HTTP and SSH flows (`go test ./...`)
- CLI tests with golden outputs
- Load tests on push/pull for packs (later phases)

## Configuration
- Environment variables via `internal/config/config.go:1`
- `GITHUT_POSTGRES_DSN`, `GITHUT_S3_REGION`, `GITHUT_S3_BUCKET`, `GITHUT_S3_ENDPOINT`, `GITHUT_HTTP_ADDR`

## Risks & Mitigations
- Pack protocol complexity: start with Smart HTTP, use git plumbing where possible
- Data consistency between local cache and S3: transactional updates and periodic reconciliation
- Security: strict RBAC defaults, rate-limits on public endpoints
- Operability: clear logs and metrics, simple health checks

## Timeline (Indicative)
- Week 1–2: Smart HTTP + local storage + auth
- Week 3: SSH server with key auth
- Week 4: CLI + migrations + permissions
- Week 5: LFS + audit + metrics
- Week 6: Backups + hardening

