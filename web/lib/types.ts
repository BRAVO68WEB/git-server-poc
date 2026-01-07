// User types
export interface UserInfo {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
}

export interface UpdateUserRequest {
  username: string;
}

export interface UpdateUserResponse {
  user: UserInfo;
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
  branch_count?: number;
  tag_count?: number;
  total_commits?: number;
  disk_usage?: number;
  language_usage_perc?: Record<string, number>;
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
  commit_hash: string;
  content: string;
  files_changed: number;
  additions: number;
  deletions: number;
  files?: DiffFile[];
}

export interface DiffFile {
  old_path: string;
  new_path: string;
  status: "added" | "deleted" | "modified" | "renamed";
  additions: number;
  deletions: number;
  patch?: string;
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

// CI/CD Types
export type CIJobStatus =
  | "pending"
  | "queued"
  | "running"
  | "success"
  | "failed"
  | "cancelled"
  | "timed_out"
  | "error";

export type CITriggerType = "push" | "tag" | "pull_request" | "manual";

export type CIRefType = "branch" | "tag";

export type CIStepType = "pre" | "exec" | "post";

export type CILogLevel = "debug" | "info" | "warning" | "error";

export interface CIJob {
  id: string;
  run_id: string;
  repository_id: string;
  commit_sha: string;
  ref_name: string;
  ref_type: CIRefType;
  trigger_type: CITriggerType;
  trigger_actor: string;
  status: CIJobStatus;
  config_path: string;
  error?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  duration_seconds?: number;
  steps?: CIJobStep[];
  artifacts?: CIArtifact[];
}

export interface CIJobStep {
  id: string;
  name: string;
  step_type: CIStepType;
  status: CIJobStatus;
  exit_code?: number;
  order: number;
  started_at?: string;
  finished_at?: string;
  duration_seconds?: number;
}

export interface CIJobLog {
  timestamp: string;
  level: CILogLevel;
  step_name?: string;
  message: string;
  sequence: number;
}

export interface CIArtifact {
  id: string;
  name: string;
  path: string;
  size: number;
  checksum: string;
  url?: string;
  created_at: string;
  expires_at?: string;
}

export interface CIJobListResponse {
  jobs: CIJob[];
  total: number;
  pagination: {
    limit: number;
    offset: number;
  };
}

export interface CIJobLogsResponse {
  job_id: string;
  logs: CIJobLog[];
  total: number;
  pagination: {
    limit: number;
    offset: number;
  };
}

export interface TriggerCIJobRequest {
  commit_sha: string;
  ref_name: string;
  ref_type: "branch" | "tag";
}

export interface TriggerCIJobResponse {
  message: string;
  job_id: string;
  run_id: string;
  status: CIJobStatus;
}

export interface CIJobEvent {
  type: "connected" | "status" | "log" | "step" | "artifact";
  job_id: string;
  data: CIJobLog | CIJob | unknown;
}
