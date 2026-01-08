# Mirror Repository Sync Feature - Complete Guide

## Overview

The Mirror Repository Sync feature provides flexible, per-repository mirroring capabilities supporting both **upstream** (pull from external source) and **downstream** (push to external destination) synchronization. Each repository can be configured independently with custom sync intervals and credentials.

## Key Features

### Bidirectional Mirroring
- **Upstream Mirror**: Pull updates from external Git repositories (GitHub, GitLab, Bitbucket, etc.)
- **Downstream Mirror**: Push updates to external Git repositories
- **Both Directions**: Simultaneously pull from one source and push to another

### Per-Repository Configuration
- ✅ Individual enable/disable per repository
- ✅ Custom sync intervals (minimum 60 seconds)
- ✅ Independent credentials for upstream and downstream
- ✅ Configurable via Web UI settings page
- ✅ No global configuration required

### Automatic Syncing
- ✅ Background cron scheduler checks every minute
- ✅ Respects each repository's individual sync interval
- ✅ Only syncs repositories that are due for sync
- ✅ Prevents concurrent syncs of the same repository

### Manual Syncing
- ✅ Trigger sync via API or Web UI
- ✅ Real-time status display
- ✅ Success/error feedback
- ✅ Shows last sync time

## Architecture

### Database Model

New fields added to `repositories` table:

```sql
-- Mirror configuration
mirror_enabled      BOOLEAN DEFAULT FALSE
mirror_direction    VARCHAR(20)  -- "upstream", "downstream", "both"
upstream_url        VARCHAR(500)
upstream_username   VARCHAR(255)
upstream_password   TEXT         -- Encrypted (TODO)
downstream_url      VARCHAR(500)
downstream_username VARCHAR(255)
downstream_password TEXT         -- Encrypted (TODO)
sync_interval       INTEGER DEFAULT 3600  -- seconds
last_synced_at      TIMESTAMP
sync_status         VARCHAR(20) DEFAULT 'idle'
sync_error          TEXT
```

### Components

#### 1. Mirror Sync Service
**File**: `internal/application/service/mirror_sync_service.go`

Handles synchronization logic:
- `SyncRepository(ctx, repoID)`: Sync single repository
- `SyncAllMirrors(ctx)`: Check and sync all due repositories
- `GetSyncStatus(ctx, repoID)`: Get current sync status

**Sync Process**:
1. Check if repository is due for sync (based on `sync_interval`)
2. Update status to "syncing"
3. If upstream configured: Fetch from upstream URL
4. If downstream configured: Push to downstream URL
5. Update sync timestamp and status
6. Store error message if failed

#### 2. Mirror Cron Service
**File**: `internal/application/service/mirror_cron_service.go`

Background scheduler:
- Runs every minute (checks repositories)
- Calculates which repositories need syncing based on their individual intervals
- Triggers sync for repositories that are due
- Skips repositories already syncing

#### 3. Git Service Extensions
**File**: `internal/infrastructure/git/git_operations.go`

New methods:
- `FetchMirror(ctx, repoPath, sourceURL)`: Pull from upstream
- `PushMirror(ctx, repoPath, destURL, username, password)`: Push to downstream

#### 4. Repository Service
**File**: `internal/application/service/repo_service.go`

New method:
- `UpdateMirrorSettings(ctx, repoID, req)`: Update mirror configuration

#### 5. HTTP Endpoints

**GET /api/v1/repos/:owner/:repo/mirror**
- Get mirror settings
- Returns configuration and sync status

**PATCH /api/v1/repos/:owner/:repo/mirror**
- Update mirror settings
- Configure upstream, downstream, intervals

**POST /api/v1/repos/:owner/:repo/sync**
- Manually trigger sync
- Returns 202 Accepted

**GET /api/v1/repos/:owner/:repo/mirror/status**
- Get current sync status
- Shows last sync time, errors

### Frontend Components

#### 1. Mirror Settings Page
**File**: `web/app/[username]/[repo]/settings/mirror/page.tsx`

Complete configuration interface:
- Enable/disable mirror sync
- Select direction (upstream, downstream, both)
- Configure upstream URL and credentials
- Configure downstream URL and credentials
- Set custom sync interval
- View sync status and errors

#### 2. Mirror Sync Button
**File**: `web/components/MirrorSyncButton.tsx`

Repository page widget:
- Shows mirror status
- Displays last sync time
- Manual sync button
- Real-time status updates

## Usage

### Via Web UI

#### 1. Configure Mirror Settings

1. Navigate to repository settings: `/{owner}/{repo}/settings/mirror`
2. Enable "Enable Mirror Sync" checkbox
3. Select mirror direction:
   - **Upstream**: Pull from external source
   - **Downstream**: Push to external destination
   - **Both**: Pull and push
4. Configure upstream settings (if applicable):
   - Source URL (e.g., `https://github.com/user/repo.git`)
   - Username (optional)
   - Password/Token (optional)
5. Configure downstream settings (if applicable):
   - Destination URL
   - Username (optional)
   - Password/Token (optional)
6. Set sync interval (seconds):
   - 3600 = 1 hour (default)
   - 7200 = 2 hours
   - 86400 = 1 day
7. Click "Save Changes"

#### 2. Manual Sync

1. Navigate to repository page
2. Find mirror sync button (if mirror enabled)
3. Click "Sync now"
4. Wait for completion

### Via API

#### Get Mirror Settings

```bash
curl http://localhost:8080/api/v1/repos/owner/repo/mirror \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "mirror_enabled": true,
  "mirror_direction": "both",
  "upstream_url": "https://github.com/user/repo.git",
  "upstream_username": "token",
  "downstream_url": "https://github.com/user/backup.git",
  "downstream_username": "token",
  "sync_interval": 3600,
  "last_synced_at": "2024-01-15T10:30:00Z",
  "sync_status": "success",
  "sync_error": ""
}
```

#### Update Mirror Settings

```bash
curl -X PATCH http://localhost:8080/api/v1/repos/owner/repo/mirror \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mirror_enabled": true,
    "mirror_direction": "upstream",
    "upstream_url": "https://github.com/user/repo.git",
    "upstream_username": "token",
    "upstream_password": "ghp_personaltoken",
    "sync_interval": 3600
  }'
```

#### Trigger Manual Sync

```bash
curl -X POST http://localhost:8080/api/v1/repos/owner/repo/sync \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "message": "Mirror sync started",
  "status": "syncing"
}
```

## Use Cases

### Use Case 1: Backup to GitHub

**Scenario**: Backup your private Git server repository to GitHub

**Configuration**:
- Direction: `downstream`
- Downstream URL: `https://github.com/user/backup-repo.git`
- Downstream Username: `token`
- Downstream Password: `ghp_yourtoken`
- Sync Interval: 3600 (1 hour)

**Result**: Every hour, changes are pushed to GitHub automatically.

### Use Case 2: Import from GitLab

**Scenario**: Keep a local mirror of a GitLab repository

**Configuration**:
- Direction: `upstream`
- Upstream URL: `https://gitlab.com/user/project.git`
- Upstream Username: `oauth2`
- Upstream Password: `gitlab-token`
- Sync Interval: 1800 (30 minutes)

**Result**: Every 30 minutes, latest changes are pulled from GitLab.

### Use Case 3: Bidirectional Sync

**Scenario**: Pull from GitHub, push to Bitbucket

**Configuration**:
- Direction: `both`
- Upstream URL: `https://github.com/user/source.git`
- Upstream Username: `token`
- Upstream Password: `ghp_token1`
- Downstream URL: `https://bitbucket.org/user/backup.git`
- Downstream Username: `user`
- Downstream Password: `app-password`
- Sync Interval: 7200 (2 hours)

**Result**: Every 2 hours, pulls from GitHub and pushes to Bitbucket.

## How It Works

### Automatic Sync Flow

```
1. Cron Scheduler (runs every minute)
   ↓
2. Query all repositories where mirror_enabled = true
   ↓
3. For each repository:
   - Check if last_synced_at + sync_interval <= now
   - Skip if already syncing
   - Skip if not due yet
   ↓
4. For repositories due for sync:
   - Update status to "syncing"
   - Spawn goroutine for async sync
   ↓
5. In background:
   - If upstream: git fetch origin '+refs/*:refs/*'
   - If downstream: git push downstream '+refs/*:refs/*'
   ↓
6. Update last_synced_at, sync_status, sync_error
```

### Manual Sync Flow

```
User clicks "Sync now"
   ↓
POST /api/v1/repos/:owner/:repo/sync
   ↓
Validate: authenticated, owner/admin, mirror enabled
   ↓
Check: not already syncing
   ↓
Trigger sync in background
   ↓
Return 202 Accepted immediately
   ↓
Background: perform sync, update status
```

## Database Migration

Required migration:

```sql
-- Add mirror columns
ALTER TABLE repositories 
ADD COLUMN mirror_enabled BOOLEAN DEFAULT FALSE,
ADD COLUMN mirror_direction VARCHAR(20),
ADD COLUMN upstream_url VARCHAR(500),
ADD COLUMN upstream_username VARCHAR(255),
ADD COLUMN upstream_password TEXT,
ADD COLUMN downstream_url VARCHAR(500),
ADD COLUMN downstream_username VARCHAR(255),
ADD COLUMN downstream_password TEXT,
ADD COLUMN sync_interval INTEGER DEFAULT 3600,
ADD COLUMN last_synced_at TIMESTAMP,
ADD COLUMN sync_status VARCHAR(20) DEFAULT 'idle',
ADD COLUMN sync_error TEXT;

-- Optional: Add index for performance
CREATE INDEX idx_repositories_mirror_enabled 
ON repositories(mirror_enabled) 
WHERE mirror_enabled = TRUE;
```

## Authentication

### GitHub
```
Username: token
Password: ghp_personalAccessToken
Scopes: repo (full control)
```

### GitLab
```
Username: oauth2
Password: gitlab-personal-token
Scopes: read_repository, write_repository
```

### Bitbucket
```
Username: bitbucket-username
Password: app-password
Permissions: Repository Read, Repository Write
```

## Monitoring

### Logs

**Sync Start**:
```
INFO  Performing mirror sync repo_id=... direction=upstream
```

**Upstream Success**:
```
INFO  Upstream sync completed repo_id=...
```

**Downstream Success**:
```
INFO  Downstream sync completed repo_id=...
```

**Sync Failure**:
```
ERROR Mirror sync failed error=... repo_id=...
```

**Cron Check**:
```
DEBUG Checking mirror repositories for sync
INFO  Mirror sync check completed total_mirrors=10 synced=3 skipped=2 errors=0
```

### Metrics

Check sync status via API:
```bash
curl http://localhost:8080/api/v1/repos/owner/repo/mirror/status \
  -H "Authorization: Bearer $TOKEN"
```

## Troubleshooting

### Sync Fails with "Authentication Failed"

**Cause**: Invalid or expired credentials

**Solution**:
1. Go to repository mirror settings
2. Update username/password
3. For GitHub: Use Personal Access Token
4. For GitLab: Use Project Access Token
5. Ensure token has read/write permissions

### Sync Status Stuck on "Syncing"

**Cause**: Previous sync crashed

**Solution**:
```sql
UPDATE repositories 
SET sync_status = 'idle' 
WHERE id = 'repo-id';
```

### Repository Not Syncing Automatically

**Checks**:
1. Verify mirror is enabled: `mirror_enabled = true`
2. Check sync interval: ensure it's >= 60 seconds
3. Check last sync time: `last_synced_at`
4. Calculate next sync: `last_synced_at + sync_interval`
5. Verify cron service is running: check logs for "Checking mirror repositories"

### Downstream Push Fails

**Common Issues**:
1. **No write access**: Ensure token has push permissions
2. **Protected branches**: Cannot push to protected branches
3. **Force push disabled**: Mirror uses force push
4. **URL mismatch**: Verify downstream URL is correct

**Solution**: Check credentials, permissions, and repository settings on destination.

## Security Considerations

### Credential Storage

**Current**: Passwords stored in plaintext (TODO: encrypt)

**Recommendation**:
- Encrypt passwords before storing
- Use application-level encryption
- Consider using secrets management service

### Access Control

- Only repository owners and admins can update mirror settings
- Passwords not exposed in API responses
- API requires authentication

### Best Practices

1. **Use Tokens**: Never use actual passwords
2. **Minimal Permissions**: Grant only required scopes
3. **Rotate Regularly**: Update tokens periodically
4. **Monitor**: Check sync logs for unauthorized access
5. **Private Repos**: Mark mirror repositories as private

## Performance Considerations

### Sync Intervals

**Recommendations**:
- **Frequently Updated**: 1800s (30 min) to 3600s (1 hour)
- **Moderately Active**: 7200s (2 hours) to 14400s (4 hours)
- **Archived/Stable**: 86400s (1 day) or more

### Concurrent Syncs

- No global limit currently
- Each repository syncs independently
- Consider adding `max_concurrent_syncs` if needed

### Network Bandwidth

- Large repositories consume bandwidth
- Consider sync interval based on repo size
- Monitor network usage during peak times

## Future Enhancements

### Planned Features

1. **Credential Encryption**: Encrypt stored passwords
2. **Selective Sync**: Choose specific branches/tags
3. **Webhook Triggers**: Sync on push notifications
4. **Sync Scheduling**: Custom schedules (cron expressions)
5. **Partial Sync**: Shallow clones for large repos
6. **Sync Metrics**: Track bytes transferred, duration
7. **Notifications**: Email/Slack on sync failures
8. **Sync Queue**: Priority queue with retry logic
9. **Branch Mapping**: Map different branch names
10. **Conflict Resolution**: Handle merge conflicts

### API Enhancements

1. **Batch Operations**: Update multiple repositories
2. **Sync History**: View past sync operations
3. **Sync Preview**: Dry-run to see changes
4. **Cancel Sync**: Stop ongoing sync

## Testing

### Test Upstream Mirror

```bash
# 1. Import repository with mirror
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-mirror",
    "clone_url": "https://github.com/octocat/Hello-World.git",
    "mirror": true
  }'

# 2. Check mirror settings
curl http://localhost:8080/api/v1/repos/user/test-mirror/mirror \
  -H "Authorization: Bearer $TOKEN"

# 3. Trigger manual sync
curl -X POST http://localhost:8080/api/v1/repos/user/test-mirror/sync \
  -H "Authorization: Bearer $TOKEN"

# 4. Check status
curl http://localhost:8080/api/v1/repos/user/test-mirror/mirror/status \
  -H "Authorization: Bearer $TOKEN"
```

### Test Downstream Mirror

```bash
# 1. Create repository
# 2. Configure downstream mirror via UI
# 3. Make changes to repository
git clone http://localhost:8080/user/repo.git
cd repo
echo "test" > test.txt
git add test.txt
git commit -m "test"
git push

# 4. Wait for sync or trigger manually
curl -X POST http://localhost:8080/api/v1/repos/user/repo/sync \
  -H "Authorization: Bearer $TOKEN"

# 5. Verify changes in downstream repository
git clone https://github.com/user/downstream-repo.git
cd downstream-repo
cat test.txt  # Should see "test"
```

## Summary

The Mirror Sync feature provides powerful, flexible repository synchronization:

- ✅ **Upstream**: Pull from external sources
- ✅ **Downstream**: Push to external destinations
- ✅ **Both**: Bidirectional mirroring
- ✅ **Per-Repository**: Individual configuration
- ✅ **Automatic**: Background syncing at custom intervals
- ✅ **Manual**: On-demand sync trigger
- ✅ **Web UI**: Easy configuration interface
- ✅ **API**: Programmatic control

Perfect for backups, mirroring open-source projects, multi-site synchronization, and disaster recovery.

---

**Version**: 2.0.0  
**Last Updated**: 2024-01-15  
**Status**: Complete