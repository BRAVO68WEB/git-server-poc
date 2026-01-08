# Mirror Sync Feature - Changes Summary

## Overview

This document summarizes all changes made to implement the automatic and manual mirror repository synchronization feature.

## Changes Made

### Backend Changes

#### 1. Domain Models

**File**: `internal/domain/models/repository.go`

**Added Fields**:
```go
IsMirror      bool       `json:"is_mirror" gorm:"default:false"`
MirrorURL     string     `json:"mirror_url,omitempty"`
LastSyncedAt  *time.Time `json:"last_synced_at,omitempty"`
SyncStatus    string     `json:"sync_status,omitempty"` // "idle", "syncing", "success", "failed"
SyncError     string     `json:"sync_error,omitempty"`
```

**Added Methods**:
- `CanSync() bool`: Returns true if repository can be synced
- `IsSyncing() bool`: Returns true if sync is in progress

#### 2. Domain Repository Interface

**File**: `internal/domain/repository/repo_repository.go`

**Added Methods**:
```go
FindAllMirrors(ctx context.Context) ([]*models.Repository, error)
```

#### 3. Repository Implementation

**File**: `internal/infrastructure/repository/repo_repo_impl.go`

**Implemented**:
- `FindAllMirrors()`: Queries all repositories where `is_mirror = true`

#### 4. Git Service Interface

**File**: `internal/domain/service/git_service.go`

**Added Methods**:
```go
FetchMirror(ctx context.Context, repoPath, sourceURL string) error
```

#### 5. Git Operations Implementation

**File**: `internal/infrastructure/git/git_operations.go`

**Implemented**:
- `FetchMirror()`: Performs git fetch with mirror refspec `+refs/*:refs/*`
- Handles "already up to date" gracefully
- Force updates all refs from source

#### 6. Mirror Sync Service (NEW)

**File**: `internal/application/service/mirror_sync_service.go`

**Created**: Complete service for mirror synchronization

**Methods**:
- `SyncRepository(ctx, repoID)`: Triggers sync for single repository
- `SyncAllMirrors(ctx)`: Syncs all mirror repositories
- `GetSyncStatus(ctx, repoID)`: Returns sync status
- `performSync(ctx, repo)`: Internal sync implementation (runs in goroutine)

**Features**:
- Prevents concurrent syncs of same repository
- Updates sync status in real-time
- Stores error messages on failure
- Background processing with goroutines

#### 7. Mirror Cron Service (NEW)

**File**: `internal/application/service/mirror_cron_service.go`

**Created**: Scheduled automatic syncing

**Methods**:
- `Start()`: Starts the cron scheduler
- `Stop()`: Gracefully stops scheduler
- `IsRunning()`: Check if running
- `SetInterval(duration)`: Update sync interval
- `GetInterval()`: Get current interval

**Features**:
- Configurable sync interval (default: 1 hour)
- Runs initial sync on start
- Ticker-based scheduling
- Graceful shutdown with WaitGroup
- Thread-safe with mutex

#### 8. Repository Service Updates

**File**: `internal/application/service/repo_service.go`

**Modified**: `ImportRepository()`
- Sets `IsMirror` field when mirror mode is enabled
- Sets `MirrorURL` to clone URL
- Initializes sync status to "idle"
- Sets `LastSyncedAt` after successful import
- Sets initial sync status to "success"

#### 9. HTTP Handler Updates

**File**: `internal/transport/http/handler/repo_handler.go`

**Modified**:
- Added `mirrorSyncService` to handler struct
- Updated `NewRepoHandler()` to accept mirror sync service

**Added Endpoints**:
- `SyncMirror()`: POST `/api/v1/repos/:owner/:repo/sync`
- `GetMirrorStatus()`: GET `/api/v1/repos/:owner/:repo/mirror/status`

**Added Import**: `github.com/google/uuid`

#### 10. Router Updates

**File**: `internal/transport/http/router/repo_router.go`

**Added Routes**:
```go
repoRoutes.POST("/sync", authMiddleware.RequireAuth(), h.SyncMirror)
repoRoutes.GET("/mirror/status", authMiddleware.Authenticate(), h.GetMirrorStatus)
```

**Added OpenAPI Documentation**:
- Sync mirror endpoint documentation
- Get mirror status endpoint documentation

#### 11. DTO Updates

**File**: `internal/application/dto/repo_dto.go`

**Modified**: `RepoResponse`
```go
IsMirror      bool       `json:"is_mirror"`
MirrorURL     string     `json:"mirror_url,omitempty"`
LastSyncedAt  *time.Time `json:"last_synced_at,omitempty"`
SyncStatus    string     `json:"sync_status,omitempty"`
```

**Updated**: `RepoFromModel()` to populate new mirror fields

#### 12. Configuration

**File**: `internal/config/config.go`

**Added Config Section**:
```go
type MirrorConfig struct {
    Enabled      bool          `mapstructure:"enabled"`
    SyncInterval time.Duration `mapstructure:"sync_interval"`
}
```

**Added to Main Config**:
```go
Mirror MirrorConfig `mapstructure:"mirror"`
```

**Added Defaults**:
```go
v.SetDefault("mirror.enabled", true)
v.SetDefault("mirror.sync_interval", "1h")
```

**Added Import**: `time` package

#### 13. Dependency Injection

**File**: `internal/injectable/dependencies.go`

**Added Services**:
```go
MirrorSyncService *service.MirrorSyncService
MirrorCronService *service.MirrorCronService
```

**Initialization**:
- Creates `MirrorSyncService` instance
- Creates `MirrorCronService` with configured interval
- Starts cron service if `mirror.enabled` is true
- Logs initialization status

### Frontend Changes

#### 1. TypeScript Types

**File**: `web/lib/types.ts`

**Modified**: `RepoResponse` interface
```typescript
is_mirror: boolean;
mirror_url?: string;
last_synced_at?: string;
sync_status?: string;
```

#### 2. API Client

**File**: `web/lib/api.ts`

**Added Functions**:
```typescript
export async function syncMirrorRepository(
  owner: string,
  repo: string,
): Promise<{ message: string; status: string }>

export async function getMirrorStatus(
  owner: string,
  repo: string,
): Promise<SyncStatus>
```

#### 3. Mirror Sync Button Component (NEW)

**File**: `web/components/MirrorSyncButton.tsx`

**Created**: Complete UI component for mirror sync

**Features**:
- Displays mirror status with animated icon
- Shows formatted last sync time ("2m ago", "5h ago", etc.)
- Manual sync button
- Loading/syncing states
- Success feedback (3 seconds)
- Error display
- Only renders for mirror repositories
- Callback on sync completion for refreshing parent component

**Visual States**:
- Idle: Shows last sync time, "Sync now" button
- Syncing: Spinning icon, disabled button
- Success: Green styling, "Synced!" message
- Failed: Red styling, error message

### Database Changes

**Required Migration**:
```sql
ALTER TABLE repositories 
ADD COLUMN is_mirror BOOLEAN DEFAULT FALSE,
ADD COLUMN mirror_url VARCHAR(500),
ADD COLUMN last_synced_at TIMESTAMP,
ADD COLUMN sync_status VARCHAR(20) DEFAULT 'idle',
ADD COLUMN sync_error TEXT;

-- Optional index for performance
CREATE INDEX idx_repositories_is_mirror 
ON repositories(is_mirror) 
WHERE is_mirror = TRUE;
```

### Configuration Changes

**File**: `config.yaml` (or environment variables)

**New Section**:
```yaml
mirror:
  enabled: true          # Enable automatic mirror syncing
  sync_interval: 1h      # Sync interval (30m, 1h, 2h, etc.)
```

**Environment Variables**:
- `STASIS_MIRROR_ENABLED`: Enable/disable (default: true)
- `STASIS_MIRROR_SYNC_INTERVAL`: Interval (default: 1h)

## API Endpoints

### 1. Sync Mirror Repository

**Endpoint**: `POST /api/v1/repos/:owner/:repo/sync`

**Authentication**: Required  
**Authorization**: Owner or Admin

**Response** (202 Accepted):
```json
{
  "message": "Mirror sync started",
  "status": "syncing"
}
```

**Errors**:
- `400`: Not a mirror repository
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Repository not found
- `500`: Internal error

### 2. Get Mirror Status

**Endpoint**: `GET /api/v1/repos/:owner/:repo/mirror/status`

**Authentication**: Required  
**Authorization**: Can access repository

**Response** (200 OK):
```json
{
  "is_mirror": true,
  "mirror_url": "https://github.com/user/repo.git",
  "last_synced_at": "2024-01-15T10:30:00Z",
  "sync_status": "success",
  "sync_error": ""
}
```

## File Summary

### New Files Created

1. `internal/application/service/mirror_sync_service.go` - Sync service
2. `internal/application/service/mirror_cron_service.go` - Cron scheduler
3. `web/components/MirrorSyncButton.tsx` - UI component
4. `docs/MIRROR_SYNC_FEATURE.md` - Feature documentation
5. `docs/MIRROR_SYNC_CHANGES.md` - This file

### Modified Files

**Backend**:
1. `internal/domain/models/repository.go` - Added mirror fields
2. `internal/domain/repository/repo_repository.go` - Added FindAllMirrors
3. `internal/infrastructure/repository/repo_repo_impl.go` - Implemented FindAllMirrors
4. `internal/domain/service/git_service.go` - Added FetchMirror interface
5. `internal/infrastructure/git/git_operations.go` - Implemented FetchMirror
6. `internal/application/service/repo_service.go` - Updated ImportRepository
7. `internal/application/dto/repo_dto.go` - Added mirror fields to response
8. `internal/transport/http/handler/repo_handler.go` - Added sync endpoints
9. `internal/transport/http/router/repo_router.go` - Added routes
10. `internal/config/config.go` - Added mirror config
11. `internal/injectable/dependencies.go` - Added services

**Frontend**:
1. `web/lib/types.ts` - Added mirror fields
2. `web/lib/api.ts` - Added sync functions

## Key Features Implemented

1. ✅ Automatic mirror syncing with configurable interval
2. ✅ Manual sync trigger via API and UI
3. ✅ Real-time sync status tracking
4. ✅ Last sync timestamp display
5. ✅ Error handling and reporting
6. ✅ Concurrent sync prevention
7. ✅ Background processing
8. ✅ Graceful cron scheduler
9. ✅ Web UI component with visual feedback
10. ✅ Comprehensive logging
11. ✅ OpenAPI documentation
12. ✅ Configuration via file or environment

## Architecture Principles

- **Clean Architecture**: Separation of concerns maintained
- **Dependency Injection**: Services injected through constructors
- **Background Processing**: Non-blocking sync operations
- **Concurrent Safety**: Mutex locks and status checks
- **Error Handling**: Comprehensive error tracking and logging
- **Configurability**: Runtime configuration via config file or env vars
- **Extensibility**: Easy to add features like webhooks, scheduling, etc.

## How It Works

### Automatic Sync Flow

1. **Startup**: Cron service starts if `mirror.enabled = true`
2. **Initial Sync**: Runs immediately on start
3. **Scheduled Sync**: Ticker fires at configured interval
4. **Query Mirrors**: Finds all repositories where `is_mirror = true`
5. **Iterate**: For each mirror (skip if already syncing):
   - Update status to "syncing"
   - Spawn goroutine for sync
   - Continue to next (non-blocking)
6. **Background Sync**: Each goroutine:
   - Calls `FetchMirror()` to update refs
   - Updates timestamp and status
   - Stores error if failed
7. **Repeat**: Next interval, process repeats

### Manual Sync Flow

1. **User Action**: Clicks "Sync now" button
2. **API Call**: `POST /repos/:owner/:repo/sync`
3. **Validation**: Check auth, ownership, is mirror, not syncing
4. **Trigger**: Call `SyncRepository()` service
5. **Background**: Sync runs in goroutine
6. **Response**: Returns 202 immediately
7. **UI Update**: Button shows "Syncing..."
8. **Completion**: Status updated, UI refreshes

## Testing

### Manual Testing

```bash
# 1. Import a mirror
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-mirror",
    "clone_url": "https://github.com/octocat/Hello-World.git",
    "mirror": true
  }'

# 2. Check status
curl http://localhost:8080/api/v1/repos/user/test-mirror/mirror/status \
  -H "Authorization: Bearer $TOKEN"

# 3. Trigger manual sync
curl -X POST http://localhost:8080/api/v1/repos/user/test-mirror/sync \
  -H "Authorization: Bearer $TOKEN"

# 4. Wait and check status again
sleep 5
curl http://localhost:8080/api/v1/repos/user/test-mirror/mirror/status \
  -H "Authorization: Bearer $TOKEN"
```

## Dependencies

No new external dependencies added. Uses existing:
- `github.com/go-git/go-git/v5` - Git operations
- Standard library `time`, `sync`, `context` - Cron and concurrency

## Performance Considerations

- **Goroutines**: Each sync runs in separate goroutine
- **Non-blocking**: Syncs don't block main thread or other syncs
- **Configurable**: Interval can be adjusted based on load
- **Indexed Query**: Optional index on `is_mirror` for faster queries

## Security

- **Authentication Required**: All endpoints require valid token
- **Authorization**: Only owner/admin can trigger sync
- **No Credential Storage**: Credentials only used during import
- **Input Validation**: Repository validation before sync

## Future Enhancements

1. **Webhook Integration**: Sync on source repository push
2. **Selective Sync**: Choose specific branches/tags
3. **Sync Queue**: Priority queue with retry logic
4. **Credential Encryption**: Store encrypted credentials for private mirrors
5. **Sync Timeout**: Maximum time limit
6. **Sync Metrics**: Track bytes transferred, duration, etc.
7. **Notification**: Alert on sync failures
8. **Sync History**: View past sync operations

## Related Documentation

- [Mirror Sync Feature](./MIRROR_SYNC_FEATURE.md) - Complete documentation
- [Import Feature](./IMPORT_FEATURE.md) - Repository import
- [Import Quick Start](./IMPORT_QUICK_START.md) - Getting started

---

**Version**: 1.0.0  
**Date**: 2024-01-15  
**Status**: Complete