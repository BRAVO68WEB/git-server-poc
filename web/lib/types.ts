// User types
export interface UserInfo {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
}

// OIDC types
export interface OIDCConfigResponse {
  oidc_enabled: boolean;
  oidc_initialized: boolean;
}

export interface OIDCCallbackResponse {
  token: string;
  user: UserInfo;
  message: string;
}

export interface OIDCLogoutResponse {
  message: string;
  logout_url?: string;
}

// Repository types
export interface CreateRepoRequest {
  name: string;
  description?: string;
  is_private?: boolean;
}

export interface UpdateRepoRequest {
  description?: string;
  is_private?: boolean;
}

export interface RepoResponse {
  id: string;
  name: string;
  owner: string;
  owner_id: string;
  is_private: boolean;
  description: string;
  clone_url: string;
  ssh_url: string;
  git_path?: string;
  created_at: string;
  updated_at: string;
}

export interface RepoListResponse {
  repositories: RepoResponse[];
  total: number;
}

export interface PublicRepoListResponse {
  repositories: RepoResponse[];
  page: number;
  per_page: number;
  total: number;
}

export interface RepoStats {
  commits: number;
  branches: number;
  tags: number;
  contributors: number;
  size_bytes: number;
}

// Branch types
export interface BranchRequest {
  name: string;
  commit_hash: string;
}

export interface BranchResponse {
  name: string;
  hash: string;
  is_head: boolean;
}

export interface BranchListResponse {
  branches: BranchResponse[];
  total: number;
}

// Tag types
export interface TagRequest {
  name: string;
  commit_hash: string;
  message?: string;
}

export interface TagResponse {
  name: string;
  hash: string;
  message?: string;
  tagger?: string;
  is_annotated: boolean;
}

export interface TagListResponse {
  tags: TagResponse[];
  total: number;
}

// Error types
export interface ErrorResponse {
  error: string;
  message: string;
  details?: string;
}

// Legacy types for existing components (tree, blob, commits views)
export interface Repo {
  owner: string;
  name: string;
  description: string;
  visibility: string;
}

export interface FileEntry {
  mode: string;
  type: string;
  hash: string;
  name: string;
  path: string;
  size?: number;
}

export interface Commit {
  hash: string;
  short_hash: string;
  author: string;
  author_email: string;
  author_date: string;
  committer: string;
  committer_email: string;
  committer_date: string;
  message: string;
  parent_hashes: string[];
}

export interface CommitListResponse {
  commits: Commit[];
  total: number;
  ref: string;
}

export interface TreeResponse {
  entries: FileEntry[];
  path: string;
  ref: string;
  total: number;
}

export interface FileContentResponse {
  path: string;
  name: string;
  size: number;
  hash: string;
  content: string;
  is_binary: boolean;
  encoding: string;
  ref: string;
}

export interface Branch {
  name: string;
  is_head: boolean;
}

export interface Diff {
  content: string;
}

export interface BlameLine {
  line_no: number;
  commit: string;
  author: string;
  email?: string;
  date: string;
  content: string;
}

export interface BlameResponse {
  blame: BlameLine[];
  path: string;
  ref: string;
  total: number;
}

// API Response wrapper for success messages
export interface SuccessResponse {
  message: string;
}

// Auth token storage
export interface AuthState {
  token: string | null;
  user: UserInfo | null;
  isAuthenticated: boolean;
}

// SSH Key types
export interface SSHKeyInfo {
  id: string;
  title: string;
  fingerprint: string;
  key_type?: string;
  last_used_at?: string;
  created_at: string;
}

export interface AddSSHKeyRequest {
  title: string;
  key: string;
}

export interface AddSSHKeyResponse {
  key: SSHKeyInfo;
  message: string;
}

export interface ListSSHKeysResponse {
  keys: SSHKeyInfo[];
  total: number;
}

// Personal Access Token types
export interface TokenInfo {
  id: string;
  name: string;
  token_hint: string;
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  created_at: string;
}

export interface CreateTokenRequest {
  name: string;
  scopes?: string[];
  expires_at?: string;
}

export interface CreateTokenResponse {
  token: string;
  token_info: TokenInfo;
  message: string;
}

export interface ListTokensResponse {
  tokens: TokenInfo[];
  total: number;
}
