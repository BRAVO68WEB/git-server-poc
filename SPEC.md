# Detailed Specification

## Protocols
### Smart HTTP
- Endpoints:
  - `POST /git/{owner}/{repo}/git-upload-pack`
  - `POST /git/{owner}/{repo}/git-receive-pack`
- Content Types:
  - Request: `application/x-git-upload-pack-request`, `application/x-git-receive-pack-request`
  - Response: `application/x-git-upload-pack-result`, `application/x-git-receive-pack-result`
- Behavior:
  - Support capability negotiation and sideband channels
  - Enforce RBAC: read requires `pull`, write requires `push`
  - Auth: `Authorization: Bearer <token>` for write operations
- Implementation:
  - HTTP handlers within `internal/git/http.go` (to be added)
  - Route wiring in `cmd/serve/main.go:1`
  - Repo resolution via DB and local cache

### SSH
- Server:
  - Listen on configurable address `GITHUT_SSH_ADDR` (to be added)
  - Use `golang.org/x/crypto/ssh`
- Authentication:
  - Public key auth; keys stored in `ssh_keys` table
  - Authorize user to repo via permissions join
- Commands:
  - `git-upload-pack '/repos/{owner}/{repo}.git'`
  - `git-receive-pack '/repos/{owner}/{repo}.git'`
- Implementation:
  - `internal/git/ssh.go` (to be added)
  - Permissions checked before invoking git service

## Storage
### Local Cache
- Directory: `data/repos/{owner}/{repo}.git`
- Contains `.git` object directory and packfiles
- Used for protocol operations; authoritative store is S3

### S3 Mapping
- Bucket: `GITHUT_S3_BUCKET`
- Key layout:
  - `repos/{owner}/{repo}.git/objects/{dir}/{file}`
  - `repos/{owner}/{repo}.git/refs/{...}`
  - `lfs/{owner}/{repo}/{oid}`
- Consistency:
  - Write-through on push: update local, then sync to S3
  - Background reconciliation job to detect drift
- Implementation:
  - `internal/storage/s3.go:1` replaces stubs with AWS SDK v2

## Database
### Tables
- `users`:
  - `id BIGSERIAL PK`
  - `username TEXT UNIQUE NOT NULL`
  - `email TEXT UNIQUE NOT NULL`
  - `role TEXT NOT NULL` (`admin`, `maintainer`, `developer`, `reader`)
  - `disabled BOOLEAN DEFAULT FALSE`
  - `created_at TIMESTAMPTZ DEFAULT now()`
- `ssh_keys`:
  - `id BIGSERIAL PK`
  - `user_id BIGINT REFERENCES users(id)`
  - `name TEXT NOT NULL`
  - `pubkey TEXT NOT NULL`
  - `fingerprint TEXT UNIQUE`
  - `created_at TIMESTAMPTZ DEFAULT now()`
- `repos`:
  - `id BIGSERIAL PK`
  - `owner_id BIGINT REFERENCES users(id)`
  - `name TEXT NOT NULL`
  - `visibility TEXT NOT NULL` (`private`, `internal`, `public`)
  - `default_branch TEXT`
  - `archived BOOLEAN DEFAULT FALSE`
  - `created_at TIMESTAMPTZ DEFAULT now()`
  - `deleted_at TIMESTAMPTZ`
  - `UNIQUE(owner_id, name)`
- `repo_members`:
  - `repo_id BIGINT REFERENCES repos(id)`
  - `user_id BIGINT REFERENCES users(id)`
  - `role TEXT NOT NULL` (`maintainer`, `developer`, `reader`)
  - `PRIMARY KEY (repo_id, user_id)`
- `audit_logs`:
  - `id BIGSERIAL PK`
  - `actor_id BIGINT REFERENCES users(id)`
  - `action TEXT NOT NULL`
  - `repo_id BIGINT REFERENCES repos(id)`
  - `ip INET`
  - `ts TIMESTAMPTZ DEFAULT now()`
  - `meta JSONB`

### Indices
- `users(username)`, `users(email)`
- `repos(owner_id, name)`, `repos(visibility)`
- `repo_members(repo_id)`, `repo_members(user_id)`
- `audit_logs(repo_id)`, `audit_logs(actor_id)`, `audit_logs(ts)`

### Migrations
- Location: `migrations/`
- Tooling: plain SQL files with incremental versioning
- First migration creates all tables and indices

## Manage CLI
### Users
- `githut manage users create --username <u> --email <e> --role <r>`
- `githut manage users delete --username <u>`
- `githut manage users list`
- `githut manage users set-role --username <u> --role <r>`

### Repos
- `githut manage repos create --owner <u> --name <n> --visibility <v>`
- `githut manage repos delete --owner <u> --name <n>`
- `githut manage repos list --owner <u>`
- `githut manage repos fork --source <u>/<n> --dest <u>/<n>`

### Admin
- `githut manage admin settings`
- `githut manage admin backup --owner <u> --name <n>`

### Stats
- `githut manage stats --owner <u> --name <n>`
- Output: commits, pushes/pulls, storage size, contributors

## Configuration
- Environment variables (see `internal/config/config.go:1`):
  - `GITHUT_POSTGRES_DSN`: PostgreSQL DSN
  - `GITHUT_S3_REGION`: S3 region
  - `GITHUT_S3_BUCKET`: S3 bucket name
  - `GITHUT_S3_ENDPOINT`: Custom S3 endpoint (optional)
  - `GITHUT_HTTP_ADDR`: HTTP bind address
  - `GITHUT_SSH_ADDR`: SSH bind address (planned)
  - `GITHUT_AUTH_TOKEN`: HTTP token for write operations (planned)

## RBAC & Permissions
- Roles:
  - `admin`: full access
  - `maintainer`: manage repo settings, push/pull
  - `developer`: push/pull
  - `reader`: pull
- Enforcement:
  - HTTP and SSH gate checks before service invocation
  - Repo visibility respected (`private`, `internal`, `public`)

## Security
- Store SSH keys with fingerprints
- Token-based auth for HTTP pushes
- Rate limiting on protocol endpoints
- Input validation and canonicalization of repo paths

## Observability
- `GET /healthz` in `cmd/serve/main.go:1`
- Add `/readyz` for readiness
- Metrics: request counts, durations, errors
- Structured logs with request IDs

## Errors
- HTTP: consistent JSON error format where applicable
- SSH: exit status and message propagation
- CLI: clear error messages and non-zero exit codes

## Backups
- Archive repository directory to S3
- Retention policy configurable
- Restore command recreates local cache and refs

## Testing
- Unit tests for storage and DB layers
- Integration tests for Smart HTTP/SSH flows
- CLI tests with sample DB fixtures

## Open Questions
- Token issuance for HTTP: static token vs OAuth2 integration
- S3-only repo operations vs mandatory local cache
- Multi-tenant namespace and org support

