Ah, got it! You want both **Frontend URLs** (web interface) and **API URLs** for Git operations. Here's a comprehensive list:

## Frontend URLs (Web Interface)

### **Repository Views**

**Repository home:**
```
https://github.com/<owner>/<repo>
```

**View specific branch:**
```
https://github.com/<owner>/<repo>/tree/<branch-name>
```

**View specific commit:**
```
https://github.com/<owner>/<repo>/tree/<commit-sha>
```

**View folder on a branch:**
```
https://github.com/<owner>/<repo>/tree/<branch-name>/<path-to-folder>
```

**View folder at a specific commit:**
```
https://github.com/<owner>/<repo>/tree/<commit-sha>/<path-to-folder>
```

**View file on a branch:**
```
https://github.com/<owner>/<repo>/blob/<branch-name>/<path-to-file>
```

**View file at a specific commit:**
```
https://github.com/<owner>/<repo>/blob/<commit-sha>/<path-to-file>
```

### **Commit Views**

**List all commits (for default branch):**
```
https://github.com/<owner>/<repo>/commits
```

**List commits for a specific branch:**
```
https://github.com/<owner>/<repo>/commits/<branch-name>
```

**View specific commit details:**
```
https://github.com/<owner>/<repo>/commit/<commit-sha>
```

**Compare commits/branches:**
```
https://github.com/<owner>/<repo>/compare/<base>...<head>
```

### **Branch and Tag Views**

**List all branches:**
```
https://github.com/<owner>/<repo>/branches
```

**List all tags:**
```
https://github.com/<owner>/<repo>/tags
```

**View specific tag:**
```
https://github.com/<owner>/<repo>/releases/tag/<tag-name>
```

### **History and Blame**

**File history (commits affecting a file):**
```
https://github.com/<owner>/<repo>/commits/<branch-name>/<path-to-file>
```

**Blame view (see who changed each line):**
```
https://github.com/<owner>/<repo>/blame/<branch-name>/<path-to-file>
```

### **Raw File Content**

**Get raw file content:**
```
https://raw.githubusercontent.com/<owner>/<repo>/<branch-or-commit>/<path-to-file>
```

---

## API URLs (REST API)

### **Repository Information**

**Get repository details:**
```
GET https://api.github.com/repos/<owner>/<repo>
```

### **Branch Operations**

**List all branches:**
```
GET https://api.github.com/repos/<owner>/<repo>/branches
```

**Get specific branch:**
```
GET https://api.github.com/repos/<owner>/<repo>/branches/<branch-name>
```

**Get branch reference (Git Data API):**
```
GET https://api.github.com/repos/<owner>/<repo>/git/ref/heads/<branch-name>
```

### **Commit Operations**

**List commits:**
```
GET https://api.github.com/repos/<owner>/<repo>/commits
```

**List commits for a specific branch:**
```
GET https://api.github.com/repos/<owner>/<repo>/commits? sha=<branch-name>
```

**Get specific commit:**
```
GET https://api.github.com/repos/<owner>/<repo>/commits/<commit-sha>
```

**Get commit object (Git Data API):**
```
GET https://api.github.com/repos/<owner>/<repo>/git/commits/<commit-sha>
```

**Compare commits:**
```
GET https://api.github.com/repos/<owner>/<repo>/compare/<base>...<head>
```

### **Tree Operations (List Files/Folders)**

**Get repository tree (all files) - non-recursive:**
```
GET https://api.github.com/repos/<owner>/<repo>/git/trees/<branch-or-commit-sha>
```

**Get repository tree - recursive (all files in all subdirectories):**
```
GET https://api.github.com/repos/<owner>/<repo>/git/trees/<branch-or-commit-sha>?recursive=1
```

### **Contents Operations (Read Files/Folders)**

**Get contents of a file or directory:**
```
GET https://api.github.com/repos/<owner>/<repo>/contents/<path>
```

**Get contents for a specific branch:**
```
GET https://api.github.com/repos/<owner>/<repo>/contents/<path>?ref=<branch-name>
```

**Get contents for a specific commit:**
```
GET https://api.github.com/repos/<owner>/<repo>/contents/<path>?ref=<commit-sha>
```

### **Blob Operations (Read File Content)**

**Get blob (file content by blob SHA):**
```
GET https://api.github.com/repos/<owner>/<repo>/git/blobs/<blob-sha>
```

**Get raw file content (alternative):**
```
GET https://raw.githubusercontent.com/<owner>/<repo>/<branch-or-commit>/<path-to-file>
```

### **Tag Operations**

**List all tags:**
```
GET https://api.github.com/repos/<owner>/<repo>/tags
```

**Get tag reference:**
```
GET https://api.github.com/repos/<owner>/<repo>/git/ref/tags/<tag-name>
```

**Get tag object:**
```
GET https://api.github.com/repos/<owner>/<repo>/git/tags/<tag-sha>
```

### **Reference Operations**

**List all references (branches, tags, etc.):**
```
GET https://api.github.com/repos/<owner>/<repo>/git/refs
```

**Get specific reference:**
```
GET https://api.github.com/repos/<owner>/<repo>/git/ref/<ref-path>
```

Examples:
- `GET /repos/owner/repo/git/ref/heads/main`
- `GET /repos/owner/repo/git/ref/tags/v1.0`

---

## Clone URLs

**HTTPS clone:**
```
https://github.com/<owner>/<repo>.git
```

**SSH clone:**
```
git@github.com:<owner>/<repo>.git
```

**GitHub CLI:**
```
gh repo clone <owner>/<repo>
```

---

## Summary Table

| Operation | Frontend URL | API URL |
|-----------|-------------|---------|
| View branch | `/tree/<branch>` | `GET /branches/<branch>` |
| View commit | `/tree/<commit-sha>` or `/commit/<commit-sha>` | `GET /commits/<commit-sha>` |
| View file | `/blob/<branch-or-commit>/<path>` | `GET /contents/<path>? ref=<ref>` |
| View folder | `/tree/<branch-or-commit>/<path>` | `GET /contents/<path>?ref=<ref>` |
| List all files | `/tree/<branch>` (browse) | `GET /git/trees/<sha>?recursive=1` |
| File history | `/commits/<branch>/<path>` | `GET /commits? path=<path>` |
| Raw content | `raw.githubusercontent.com/<owner>/<repo>/<ref>/<path>` | `GET /git/blobs/<sha>` |

---

This should give you a complete reference for implementing both the web interface and API for your self-hosted Git server!  🚀