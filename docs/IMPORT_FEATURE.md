# Import Repository Feature

## Overview

The Import Repository feature allows users to import existing Git repositories from external sources (GitHub, GitLab, Bitbucket, or any Git hosting service) into the Git Server. This feature supports both regular imports and mirror repositories.

## Features

- **Import from Any Git Source**: Support for importing from GitHub, GitLab, Bitbucket, and any Git-compatible server
- **Authentication Support**: Basic authentication with username/password or personal access tokens
- **Mirror Mode**: Create mirror repositories that maintain all refs (branches and tags) from the source
- **Private Repository Support**: Import private repositories with authentication
- **Automatic Branch Detection**: Automatically detects and sets the default branch from the source repository

## Architecture

### Backend Components

#### 1. Data Transfer Objects (DTOs)

**File**: `internal/application/dto/repo_dto.go`

```go
type ImportRepoRequest struct {
    Name        string `json:"name" binding:"required,min=1,max=100"`
    Description string `json:"description" binding:"max=500"`
    IsPrivate   bool   `json:"is_private"`
    CloneURL    string `json:"clone_url" binding:"required,url"`
    Username    string `json:"username,omitempty"`
    Password    string `json:"password,omitempty"`
    Mirror      bool   `json:"mirror"`
}
```

**Validation Rules**:
- `name`: Required, 1-100 characters, alphanumeric with hyphens, underscores, periods
- `description`: Optional, max 500 characters
- `clone_url`: Required, must be a valid URL
- `username/password`: Optional, used for private repository authentication
- `mirror`: Optional boolean, defaults to false

#### 2. Application Service

**File**: `internal/application/service/repo_service.go`

**Method**: `ImportRepository`

```go
func (s *RepoService) ImportRepository(
    ctx context.Context,
    ownerID uuid.UUID,
    name, description, cloneURL, username, password string,
    isPrivate, mirror bool,
) (*models.Repository, error)
```

**Process Flow**:
1. Validate input parameters
2. Verify owner exists
3. Check if repository name is already taken
4. Clone repository from external source using Git service
5. Configure as mirror if requested
6. Detect default branch from cloned repository
7. Save repository metadata to database
8. Return repository information

#### 3. Domain Service

**File**: `internal/domain/service/git_service.go`

**Updated Interface Methods**:

```go
// CloneRepository clones a repository from source to destination
CloneRepository(ctx context.Context, source, dest, username, password string, mirror bool) error

// ConfigureMirror configures a repository as a mirror of the source
ConfigureMirror(ctx context.Context, repoPath, sourceURL string) error
```

#### 4. Git Operations Implementation

**File**: `internal/infrastructure/git/git_operations.go`

**Clone Repository**:
- Uses `go-git` library for cloning
- Supports HTTP basic authentication
- Creates bare repositories when mirror mode is enabled
- Handles authentication with personal access tokens

**Configure Mirror**:
- Sets up remote configuration for mirroring
- Configures fetch refspec to pull all refs: `+refs/*:refs/*`
- Enables mirror flag in remote configuration

#### 5. HTTP Handler

**File**: `internal/transport/http/handler/repo_handler.go`

**Endpoint**: `POST /api/v1/repos/import`

**Handler**: `ImportRepository`

**Authentication**: Required (uses auth middleware)

**Request Body**:
```json
{
  "name": "my-imported-repo",
  "description": "Imported from GitHub",
  "clone_url": "https://github.com/username/repository.git",
  "username": "git-username",
  "password": "personal-access-token",
  "is_private": true,
  "mirror": false
}
```

**Response** (201 Created):
```json
{
  "id": "uuid",
  "name": "my-imported-repo",
  "owner": "username",
  "owner_id": "uuid",
  "is_private": true,
  "description": "Imported from GitHub",
  "clone_url": "http://localhost:8080/username/my-imported-repo.git",
  "ssh_url": "git@localhost:username/my-imported-repo.git",
  "default_branch": "main",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### 6. Router

**File**: `internal/transport/http/router/repo_router.go`

**Route Registration**:
```go
repos.POST("/import", authMiddleware.RequireAuth(), h.ImportRepository)
```

**OpenAPI Documentation**: Automatically registered with tags and response schemas

### Frontend Components

#### 1. TypeScript Types

**File**: `web/lib/types.ts`

```typescript
export interface ImportRepoRequest {
  name: string;
  description?: string;
  is_private?: boolean;
  clone_url: string;
  username?: string;
  password?: string;
  mirror?: boolean;
}
```

#### 2. API Client

**File**: `web/lib/api.ts`

**Function**: `importRepository`

```typescript
export async function importRepository(
  data: ImportRepoRequest,
): Promise<RepoResponse>
```

#### 3. Import Page

**File**: `web/app/import/page.tsx`

**Features**:
- Form validation (client-side and server-side)
- URL validation
- Repository name validation
- Optional authentication fields
- Privacy toggle
- Mirror mode toggle
- Loading states
- Error handling and display

**Form Fields**:
- **Clone URL** (required): Git repository URL
- **Repository Name** (required): Name for the imported repository
- **Description** (optional): Repository description
- **Username** (optional): For private repository authentication
- **Password/Token** (optional): Personal access token or password
- **Private Repository** (checkbox): Mark repository as private
- **Mirror Repository** (checkbox): Create a mirror repository

#### 4. Header Navigation

**File**: `web/components/Header.tsx`

**Updated "New" Button**:
- Changed to dropdown menu
- Options:
  - **New Repository**: Create a new empty repository
  - **Import Repository**: Import from external Git source

## Usage

### Via Web Interface

1. **Navigate to Import Page**:
   - Click the "New" button in the header
   - Select "Import repository" from the dropdown

2. **Fill in the Form**:
   - Enter the Git clone URL (e.g., `https://github.com/user/repo.git`)
   - Choose a name for the repository
   - Add optional description
   - For private repositories:
     - Enter username
     - Enter personal access token or password
   - Choose privacy settings
   - Enable mirror mode if desired

3. **Submit**:
   - Click "Import repository"
   - Wait for the import to complete
   - Redirected to the imported repository page

### Via API

**cURL Example**:

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "my-repo",
    "description": "Imported from GitHub",
    "clone_url": "https://github.com/username/repository.git",
    "username": "username",
    "password": "ghp_personalAccessToken",
    "is_private": true,
    "mirror": false
  }'
```

**Using Personal Access Token for GitHub**:

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "my-repo",
    "clone_url": "https://github.com/username/repository.git",
    "username": "token",
    "password": "ghp_yourPersonalAccessToken",
    "is_private": false
  }'
```

## Mirror vs Regular Import

### Regular Import

- Creates a standard bare repository
- Clones all branches and tags
- Repository is independent of the source
- Updates must be pushed manually

### Mirror Import

- Creates a mirror repository
- Configured with `+refs/*:refs/*` fetch refspec
- All refs (branches, tags, etc.) are synchronized
- Remote tracking is maintained
- Can be updated with `git fetch --mirror` commands

## Authentication Methods

### 1. GitHub

Use a Personal Access Token (PAT):
- Username: `token` or your GitHub username
- Password: Your PAT (e.g., `ghp_xxxxx`)
- Scopes required: `repo` (full control of private repositories)

### 2. GitLab

Use a Personal Access Token:
- Username: Your GitLab username or `oauth2`
- Password: Your PAT
- Scopes required: `read_repository` or `api`

### 3. Bitbucket

Use App Password:
- Username: Your Bitbucket username
- Password: Your App Password
- Permissions required: Repository Read

### 4. Self-hosted Git

- Username: Your username on the Git server
- Password: Your password or token
- May vary based on server configuration

## Error Handling

### Common Errors

1. **Invalid Clone URL**:
   - Error: "Invalid clone URL format"
   - Solution: Ensure URL is properly formatted

2. **Authentication Failed**:
   - Error: "Failed to clone repository"
   - Solution: Check username/password, ensure token has proper permissions

3. **Repository Already Exists**:
   - Error: "Repository already exists"
   - Solution: Choose a different name

4. **Network Issues**:
   - Error: "Failed to clone repository"
   - Solution: Check network connectivity, verify source repository is accessible

5. **Insufficient Permissions**:
   - Error: "Failed to clone repository"
   - Solution: Ensure token/credentials have read access to the repository

## Security Considerations

1. **Credentials Storage**: Credentials are only used during the import process and are not stored
2. **HTTPS Recommended**: Use HTTPS URLs for secure data transmission
3. **Token Permissions**: Use tokens with minimal required permissions
4. **Private Repositories**: Properly set `is_private` flag for imported private repositories

## Implementation Details

### Database Schema

No changes to the database schema were required. The import feature uses the existing `repositories` table.

### Git Operations

The implementation uses the `go-git` library:
- `git.PlainClone()` for cloning repositories
- `git.CloneOptions` with authentication support
- `git.Config` for mirror configuration

### Performance Considerations

1. **Large Repositories**: Import time depends on repository size and network speed
2. **Timeouts**: Consider implementing timeout handling for very large repositories
3. **Background Jobs**: For production, consider moving imports to background jobs

## Testing

### Manual Testing

1. **Import Public Repository**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/repos/import \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "test-repo",
       "clone_url": "https://github.com/octocat/Hello-World.git"
     }'
   ```

2. **Import Private Repository**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/repos/import \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "private-repo",
       "clone_url": "https://github.com/user/private-repo.git",
       "username": "token",
       "password": "ghp_xxxxx",
       "is_private": true
     }'
   ```

3. **Create Mirror**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/repos/import \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "mirror-repo",
       "clone_url": "https://github.com/user/repo.git",
       "mirror": true
     }'
   ```

## Future Enhancements

1. **Background Import**: Move import operations to background jobs with progress tracking
2. **Webhook Integration**: Automatically sync mirror repositories when source changes
3. **Import from Archive**: Support importing from ZIP/TAR archives
4. **Import History**: Track import history and source information
5. **Batch Import**: Import multiple repositories at once
6. **Import Templates**: Save import configurations as templates
7. **SSH Support**: Add support for SSH URLs with key-based authentication
8. **Scheduled Sync**: Automatic periodic synchronization for mirror repositories

## Troubleshooting

### Import Fails Silently

Check the server logs:
```bash
docker logs git-server-dev
```

### Mirror Not Syncing

Manually fetch updates:
```bash
cd /path/to/repo.git
git fetch --mirror
```

### Authentication Issues with GitHub

1. Ensure PAT hasn't expired
2. Check PAT scopes include `repo`
3. Use `token` as username, not your GitHub username
4. Enable SSO if repository is in an organization

## Related Files

- Backend:
  - `internal/application/dto/repo_dto.go`
  - `internal/application/service/repo_service.go`
  - `internal/domain/service/git_service.go`
  - `internal/infrastructure/git/git_operations.go`
  - `internal/transport/http/handler/repo_handler.go`
  - `internal/transport/http/router/repo_router.go`

- Frontend:
  - `web/lib/types.ts`
  - `web/lib/api.ts`
  - `web/app/import/page.tsx`
  - `web/components/Header.tsx`

## License

This feature is part of the Git Server project and follows the same license.