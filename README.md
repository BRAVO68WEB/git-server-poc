# Stasis

A self-hosted Git server implementation with SSH and HTTP protocol support, built with Go.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                                      CLIENTS                                             │
├─────────────────────┬─────────────────────┬─────────────────────────────────────────────┤
│     Git CLI         │    Web Browser      │              CI Runner                       │
│  (clone/push/pull)  │   (Web Interface)   │         (Job Execution)                      │
└─────────┬───────────┴──────────┬──────────┴────────────────────┬────────────────────────┘
          │                      │                               │
          │ SSH (Port 2222)      │ HTTP (Port 80/443)            │ HTTP API
          │                      │                               │
┌─────────▼──────────────────────▼───────────────────────────────▼────────────────────────┐
│                                     NGINX                                                │
│                              (Reverse Proxy / Load Balancer)                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │
│  │ • Routes /api/* → API Backend                                                       │ │
│  │ • Routes Git Smart HTTP (info/refs, git-upload-pack, git-receive-pack) → API       │ │
│  │ • Routes User-Agent: git/* → API Backend                                            │ │
│  │ • Routes Web UI requests → Web Backend (Next.js)                                    │ │
│  │ • Rate limiting, CORS, Security headers                                             │ │
│  └─────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────┬──────────────────────────────┬────────────────────────────┘
                              │                              │
              ┌───────────────▼──────────────┐   ┌───────────▼───────────────┐
              │      API SERVER (Go)         │   │   WEB FRONTEND (Next.js)  │
              │        Port 8080             │   │       Port 3000           │
              ├──────────────────────────────┤   ├───────────────────────────┤
              │                              │   │                           │
              │  ┌────────────────────────┐  │   │  • Dashboard              │
              │  │  HTTP Transport Layer  │  │   │  • Repository Browser     │
              │  │  (Gin Framework)       │  │   │  • User Settings          │
              │  ├────────────────────────┤  │   │  • SSH Key Management     │
              │  │  Handlers:             │  │   │  • Auth (OAuth/OIDC)      │
              │  │  • Auth Handler        │  │   │                           │
              │  │  • Repo Handler        │  │   └───────────────────────────┘
              │  │  • Git Handler         │  │
              │  │  • SSH Key Handler     │  │
              │  │  • Token Handler       │  │
              │  │  • CI Handler          │  │
              │  │  • Health Handler      │  │
              │  └────────────────────────┘  │
              │                              │
              │  ┌────────────────────────┐  │
              │  │  SSH Transport Layer   │  │◄──── SSH (Port 2222)
              │  │  (Charm/Wish)          │  │      git-upload-pack
              │  ├────────────────────────┤  │      git-receive-pack
              │  │  • Public Key Auth     │  │
              │  │  • Git Protocol Exec   │  │
              │  │  • CI Trigger on Push  │  │
              │  └────────────────────────┘  │
              │                              │
              ├──────────────────────────────┤
              │      APPLICATION LAYER       │
              │  ┌────────────────────────┐  │
              │  │  Services:             │  │
              │  │  • AuthService         │  │
              │  │  • RepoService         │  │
              │  │  • UserService         │  │
              │  │  • SSHKeyService       │  │
              │  │  • TokenService        │  │
              │  │  • CIService           │  │
              │  │  • OIDCService         │  │
              │  └────────────────────────┘  │
              │                              │
              ├──────────────────────────────┤
              │       DOMAIN LAYER           │
              │  ┌────────────────────────┐  │
              │  │  Models:               │  │
              │  │  • User                │  │
              │  │  • Repository          │  │
              │  │  • SSHKey              │  │
              │  │  • Token               │  │
              │  │  • CIJob               │  │
              │  └────────────────────────┘  │
              │                              │
              ├──────────────────────────────┤
              │    INFRASTRUCTURE LAYER      │
              │  ┌────────────────────────┐  │
              │  │  Git Operations:       │  │
              │  │  • GitProtocol         │  │──────► Repository Storage
              │  │  • GitOperations       │  │        (Filesystem/S3)
              │  ├────────────────────────┤  │
              │  │  Repositories (DB):    │  │
              │  │  • UserRepository      │  │──────► PostgreSQL
              │  │  • RepoRepository      │  │
              │  │  • SSHKeyRepository    │  │
              │  │  • TokenRepository     │  │
              │  │  • CIRepository        │  │
              │  ├────────────────────────┤  │
              │  │  Storage Backends:     │  │
              │  │  • FilesystemStorage   │  │──────► Local Disk
              │  │  • S3Storage           │  │──────► S3/MinIO
              │  ├────────────────────────┤  │
              │  │  Observability:        │  │
              │  │  • OpenTelemetry       │  │──────► OTEL Collector
              │  │  • Structured Logging  │  │
              │  └────────────────────────┘  │
              └──────────────────────────────┘
                              │
                              │
              ┌───────────────▼──────────────┐
              │         POSTGRESQL           │
              │          Port 5432           │
              ├──────────────────────────────┤
              │  Tables:                     │
              │  • users                     │
              │  • repositories              │
              │  • ssh_keys                  │
              │  • tokens                    │
              │  • ci_jobs                   │
              │  • ci_job_steps              │
              │  • ci_job_logs               │
              │  • ci_artifacts              │
              └──────────────────────────────┘
```

## Component Overview

### Transport Layer

| Component | Description |
|-----------|-------------|
| **HTTP API** | REST API built with Gin framework for web clients and API consumers |
| **SSH Server** | SSH server using Charm/Wish for Git operations (clone, push, pull) |
| **Git Protocol** | Smart HTTP protocol implementation for web-based Git operations |

### Application Services

| Service | Description |
|---------|-------------|
| **AuthService** | Handles authentication via SSH keys and HTTP tokens |
| **RepoService** | Repository CRUD operations and access control |
| **UserService** | User management and profile operations |
| **SSHKeyService** | SSH public key management for authentication |
| **TokenService** | API token generation and validation |
| **CIService** | CI/CD job triggering and status management |
| **OIDCService** | OpenID Connect integration for SSO |

### Infrastructure

| Component | Description |
|-----------|-------------|
| **PostgreSQL** | Primary database for users, repos, keys, tokens, and CI jobs |
| **Filesystem Storage** | Local disk storage for Git repositories |
| **S3 Storage** | Optional S3-compatible storage for repositories and artifacts |
| **OpenTelemetry** | Distributed tracing and observability |

## Data Flow

### Git Clone (HTTP)
```
Client → NGINX → API (Git Handler) → GitProtocol → Filesystem → Response
```

### Git Push (SSH)
```
Client → SSH Server → Auth (Public Key) → GitProtocol → Filesystem → CI Trigger → Response
```

### Web UI Request
```
Browser → NGINX → Next.js → API Calls → API Server → PostgreSQL → Response
```

### CI Job Trigger
```
Git Push → SSH/HTTP Handler → CIService → External CI Runner → Job Status Updates
```

## Directory Structure

```
git-server-poc/
├── cmd/
│   ├── cli/              # CLI management tools
│   ├── migrations/       # Database migration runner
│   └── server/           # Main server entry point
├── internal/
│   ├── application/      # Application services
│   │   ├── commands/     # Command handlers
│   │   ├── dto/          # Data transfer objects
│   │   └── service/      # Business logic services
│   ├── config/           # Configuration loading
│   ├── domain/           # Domain models and interfaces
│   │   ├── models/       # Entity definitions
│   │   ├── repository/   # Repository interfaces
│   │   └── service/      # Service interfaces
│   ├── infrastructure/   # External integrations
│   │   ├── database/     # Database connection
│   │   ├── git/          # Git protocol implementation
│   │   ├── otel/         # OpenTelemetry setup
│   │   ├── repository/   # Repository implementations
│   │   └── storage/      # Storage backends (FS/S3)
│   ├── injectable/       # Dependency injection
│   ├── server/           # Server initialization
│   └── transport/        # Transport layer
│       ├── http/         # HTTP handlers, middleware, routers
│       └── ssh/          # SSH server implementation
├── pkg/
│   ├── errors/           # Error handling utilities
│   └── logger/           # Structured logging
├── web/                  # Next.js frontend application
├── configs/              # Configuration files
├── deploy/               # Deployment configurations (nginx, etc.)
└── data/                 # Runtime data (repos, etc.)
```

## API Endpoints

### Authentication
- `POST /api/auth/login` - OAuth/OIDC login
- `POST /api/auth/callback` - OAuth callback
- `GET /api/auth/me` - Get current user

### Repositories
- `GET /api/repos` - List repositories
- `POST /api/repos` - Create repository
- `GET /api/repos/:owner/:repo` - Get repository details
- `DELETE /api/repos/:owner/:repo` - Delete repository

### Git Protocol (Smart HTTP)
- `GET /:owner/:repo/info/refs` - Advertise refs
- `POST /:owner/:repo/git-upload-pack` - Fetch/Clone
- `POST /:owner/:repo/git-receive-pack` - Push

### SSH Keys
- `GET /api/ssh-keys` - List user's SSH keys
- `POST /api/ssh-keys` - Add SSH key
- `DELETE /api/ssh-keys/:id` - Remove SSH key

### CI/CD
- `GET /api/ci/jobs` - List CI jobs
- `GET /api/ci/jobs/:id` - Get job details
- `POST /api/ci/jobs/:id/logs` - Receive job logs
- `PUT /api/ci/jobs/:id/status` - Update job status

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for development)
- Node.js 18+ (for frontend development)

### Running with Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Development

```bash
# Install dependencies
make setup

# Run API server (with hot reload)
make dev

# Run frontend
cd web && npm run dev
```

### Configuration

Configuration is managed via `configs/config.yaml` and environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GITSERVER_DATABASE_HOST` | PostgreSQL host | `localhost` |
| `GITSERVER_DATABASE_PORT` | PostgreSQL port | `5432` |
| `GITSERVER_SERVER_PORT` | HTTP server port | `8080` |
| `GITSERVER_SSH_PORT` | SSH server port | `2222` |
| `GITSERVER_STORAGE_TYPE` | Storage backend (`filesystem`/`s3`) | `filesystem` |

## License

MIT License