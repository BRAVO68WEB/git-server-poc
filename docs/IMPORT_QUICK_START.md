# Quick Start: Import Repository Feature

## Overview

This guide will help you quickly import a Git repository from GitHub, GitLab, Bitbucket, or any other Git hosting service into your Git Server.

## Prerequisites

- Access to the Git Server web interface
- Valid authentication (logged in)
- Git repository URL you want to import
- (Optional) Authentication credentials if importing a private repository

## Web UI Method

### Step 1: Navigate to Import Page

1. Log in to your Git Server
2. Click the **"New"** button in the top navigation bar
3. Select **"Import repository"** from the dropdown menu

### Step 2: Fill Out the Import Form

#### Required Fields

1. **Clone URL**: Enter the Git repository URL
   ```
   Examples:
   - https://github.com/username/repository.git
   - https://gitlab.com/username/repository.git
   - https://bitbucket.org/username/repository.git
   ```

2. **Repository Name**: Choose a name for the imported repository
   ```
   Rules:
   - Letters, numbers, hyphens, underscores, periods
   - Maximum 100 characters
   - Example: my-awesome-project
   ```

#### Optional Fields

3. **Description**: Add a description (max 500 characters)

4. **Authentication** (for private repositories):
   - **Username**: Your username or "token"
   - **Password/Token**: Personal access token or password

5. **Options**:
   - ☐ **Private repository**: Only you and collaborators can see it
   - ☐ **Mirror repository**: Keep synchronized with source

### Step 3: Submit

Click the **"Import repository"** button and wait for the import to complete. You'll be redirected to your new repository when done.

## API Method

### Import Public Repository

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-repo",
    "clone_url": "https://github.com/octocat/Hello-World.git"
  }'
```

### Import Private Repository (GitHub)

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-private-repo",
    "clone_url": "https://github.com/username/private-repo.git",
    "username": "token",
    "password": "ghp_yourPersonalAccessToken",
    "is_private": true
  }'
```

### Create Mirror Repository

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mirror-repo",
    "clone_url": "https://github.com/username/repository.git",
    "mirror": true
  }'
```

## Authentication Guide

### GitHub

1. Create a Personal Access Token (PAT):
   - Go to Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token"
   - Select scopes: `repo` (full control of private repositories)
   - Copy the token (starts with `ghp_`)

2. Use in import form:
   - **Username**: `token` (or your GitHub username)
   - **Password**: Your PAT (e.g., `ghp_xxxxxxxxxxxxx`)

### GitLab

1. Create a Personal Access Token:
   - Go to Preferences → Access Tokens
   - Create token with `read_repository` scope
   - Copy the token

2. Use in import form:
   - **Username**: Your GitLab username (or `oauth2`)
   - **Password**: Your PAT

### Bitbucket

1. Create an App Password:
   - Go to Personal settings → App passwords
   - Create password with "Repository Read" permission
   - Copy the password

2. Use in import form:
   - **Username**: Your Bitbucket username
   - **Password**: Your App Password

## Common Examples

### Example 1: Import Popular Open Source Project

```json
{
  "name": "react",
  "clone_url": "https://github.com/facebook/react.git",
  "description": "A JavaScript library for building user interfaces"
}
```

### Example 2: Import Your Private Project

```json
{
  "name": "my-project",
  "clone_url": "https://github.com/myusername/my-private-project.git",
  "username": "token",
  "password": "ghp_yourtoken",
  "is_private": true,
  "description": "My private project"
}
```

### Example 3: Create a Mirror

```json
{
  "name": "upstream-mirror",
  "clone_url": "https://github.com/organization/project.git",
  "mirror": true,
  "description": "Mirror of upstream project"
}
```

## Troubleshooting

### Import Fails

**Problem**: "Failed to clone repository"

**Solutions**:
- ✓ Check the clone URL is correct
- ✓ Verify the repository exists and is accessible
- ✓ For private repos, ensure credentials are correct
- ✓ Check your network connection
- ✓ Verify token hasn't expired

### Authentication Errors

**Problem**: "Authentication failed"

**Solutions**:
- ✓ Use personal access token, not your password
- ✓ Verify token has correct permissions/scopes
- ✓ For GitHub, use `token` as username
- ✓ Check if token has SSO authorization (for org repos)

### Repository Name Already Exists

**Problem**: "Repository already exists"

**Solutions**:
- ✓ Choose a different repository name
- ✓ Delete the existing repository first (if appropriate)

## Tips & Best Practices

1. **Use Personal Access Tokens**: Never use your actual password. Always use PATs for better security.

2. **Token Permissions**: Only grant the minimum permissions needed (typically just read access).

3. **Repository Size**: Large repositories may take several minutes to import. Be patient.

4. **Mirror Mode**: Use mirror mode if you want to keep the repository synchronized with the source.

5. **Private Repositories**: Always mark imported private repositories as private in your Git Server.

6. **Test First**: Try importing a small public repository first to ensure everything works.

## What Happens During Import?

1. **Validation**: The system validates your input (URL, name, credentials)
2. **Cloning**: Git Server clones the repository from the source
3. **Branch Detection**: Automatically detects and sets the default branch
4. **Storage**: Saves repository metadata to the database
5. **Completion**: Redirects you to the imported repository

## Next Steps

After importing:

1. **Browse Code**: Explore your imported repository
2. **Clone Locally**: Clone to your local machine
3. **Add Collaborators**: Share with team members (if private)
4. **Setup CI/CD**: Configure continuous integration
5. **Sync Mirror**: Periodically sync if it's a mirror repository

## Support

- View full documentation: `docs/IMPORT_FEATURE.md`
- Check API details: `/api-docs` endpoint
- Report issues: Contact your Git Server administrator

## Examples by Platform

### From GitHub to Git Server

```bash
# Public repo
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "hello-world", "clone_url": "https://github.com/octocat/Hello-World.git"}'

# Private repo
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-private",
    "clone_url": "https://github.com/user/private.git",
    "username": "token",
    "password": "ghp_token123",
    "is_private": true
  }'
```

### From GitLab to Git Server

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gitlab-project",
    "clone_url": "https://gitlab.com/user/project.git",
    "username": "oauth2",
    "password": "your-gitlab-token"
  }'
```

### From Bitbucket to Git Server

```bash
curl -X POST http://localhost:8080/api/v1/repos/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "bitbucket-repo",
    "clone_url": "https://bitbucket.org/user/repo.git",
    "username": "your-username",
    "password": "your-app-password"
  }'
```

---

**That's it!** You're now ready to import repositories into your Git Server. Happy coding! 🚀