import {
  UserInfo,
  OIDCConfigResponse,
  OIDCCallbackResponse,
  OIDCLogoutResponse,
  CreateRepoRequest,
  UpdateRepoRequest,
  RepoResponse,
  RepoListResponse,
  PublicRepoListResponse,
  RepoStats,
  BranchRequest,
  BranchResponse,
  BranchListResponse,
  TagRequest,
  TagResponse,
  TagListResponse,
  SuccessResponse,
  ErrorResponse,
  Repo,
  FileEntry,
  Commit,
  Branch,
  Diff,
  BlameLine,
  CommitListResponse,
  TreeResponse,
  FileContentResponse,
  SSHKeyInfo,
  AddSSHKeyRequest,
  AddSSHKeyResponse,
  ListSSHKeysResponse,
} from "./types";
import { env } from "./env";

const API_URL = env.NEXT_PUBLIC_API_URL;

// Token storage helpers (client-side only)
function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("auth_token");
}

function setToken(token: string): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("auth_token", token);
  }
}

function removeToken(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem("auth_token");
  }
}

// User info storage helpers (client-side only)
function getUserInfo(): UserInfo | null {
  if (typeof window === "undefined") return null;
  const userInfoStr = localStorage.getItem("user_info");
  if (!userInfoStr) return null;
  try {
    return JSON.parse(userInfoStr) as UserInfo;
  } catch {
    return null;
  }
}

function setUserInfo(user: UserInfo): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("user_info", JSON.stringify(user));
  }
}

function removeUserInfo(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem("user_info");
  }
}

// Request helper with authentication
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  const token = getToken();
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  if (token) {
    // All tokens are now Bearer tokens (JWT from OIDC or PAT)
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${endpoint}`, {
    ...options,
    headers,
    cache: "no-store",
  });

  if (!res.ok) {
    const errorData: ErrorResponse = await res.json().catch(() => ({
      error: "unknown_error",
      message: `Request failed with status ${res.status}`,
    }));
    throw new Error(errorData.message || "Request failed");
  }

  return res.json();
}

// ============================================================================
// Health API
// ============================================================================

export async function getHealth(): Promise<{ name: string; status: string }> {
  return apiRequest("/");
}

// ============================================================================
// OIDC Authentication API
// ============================================================================

/**
 * Get OIDC configuration status
 * Returns whether OIDC is enabled and initialized
 */
export async function getOIDCConfig(): Promise<OIDCConfigResponse> {
  return apiRequest("/api/v1/auth/oidc/config");
}

/**
 * Get the OIDC login URL
 * The user should be redirected to this URL to start the OIDC flow
 */
export function getOIDCLoginURL(): string {
  return `${API_URL}/api/v1/auth/oidc/login`;
}

/**
 * Initiate OIDC login by redirecting to the identity provider
 * This will redirect the browser to the OIDC provider's login page
 */
export function initiateOIDCLogin(): void {
  if (typeof window !== "undefined") {
    window.location.href = getOIDCLoginURL();
  }
}

/**
 * Handle OIDC callback - exchange code for token
 * This is typically called from the callback page after the OIDC provider redirects back
 * The token is automatically stored in localStorage
 */
export async function handleOIDCCallback(
  code: string,
  state: string,
): Promise<OIDCCallbackResponse> {
  // The callback is handled server-side, but we need to process the response
  // This function is called after the redirect, when we have the token in the response
  const response = await apiRequest<OIDCCallbackResponse>(
    `/api/v1/auth/oidc/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
  );

  // Store the session token
  if (response.token) {
    setToken(response.token);
  }

  // Store the user info
  if (response.user) {
    setUserInfo(response.user);
  }

  return response;
}

/**
 * Store the auth token (called from callback page)
 */
export function storeAuthToken(token: string): void {
  setToken(token);
}

/**
 * Get the current authenticated user
 */
export async function getCurrentUser(): Promise<UserInfo> {
  return apiRequest("/api/v1/auth/me");
}

/**
 * Logout the user
 * Clears the local token and optionally returns the provider logout URL
 */
export async function logout(
  redirectUri?: string,
): Promise<OIDCLogoutResponse | null> {
  try {
    const params = redirectUri
      ? `?redirect_uri=${encodeURIComponent(redirectUri)}`
      : "";
    const response = await apiRequest<OIDCLogoutResponse>(
      `/api/v1/auth/oidc/logout${params}`,
      { method: "POST" },
    );

    // Clear local token and user info
    removeToken();
    removeUserInfo();

    return response;
  } catch {
    // Even if the API call fails, clear the local token and user info
    removeToken();
    removeUserInfo();
    return null;
  }
}

/**
 * Logout locally only (clear token without calling the API)
 */
export function logoutLocal(): void {
  removeToken();
  removeUserInfo();
}

/**
 * Store user info in localStorage
 */
export function storeUserInfo(user: UserInfo): void {
  setUserInfo(user);
}

/**
 * Get stored user info from localStorage (without API call)
 */
export function getStoredUserInfo(): UserInfo | null {
  return getUserInfo();
}

/**
 * Check if the user is authenticated (has a token)
 */
export function isAuthenticated(): boolean {
  return getToken() !== null;
}

/**
 * Get the current auth token
 */
export function getAuthToken(): string | null {
  return getToken();
}

// ============================================================================
// SSH Key API
// ============================================================================

export async function listSSHKeys(): Promise<ListSSHKeysResponse> {
  return apiRequest("/api/v1/ssh-keys");
}

export async function addSSHKey(
  data: AddSSHKeyRequest,
): Promise<AddSSHKeyResponse> {
  return apiRequest("/api/v1/ssh-keys", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getSSHKey(id: string): Promise<SSHKeyInfo> {
  return apiRequest(`/api/v1/ssh-keys/${encodeURIComponent(id)}`);
}

export async function deleteSSHKey(id: string): Promise<SuccessResponse> {
  return apiRequest(`/api/v1/ssh-keys/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

// ============================================================================
// Repository API
// ============================================================================

export async function listUserRepositories(): Promise<RepoListResponse> {
  return apiRequest("/api/v1/repos");
}

export async function createRepository(
  data: CreateRepoRequest,
): Promise<RepoResponse> {
  return apiRequest("/api/v1/repos", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function listPublicRepositories(
  page: number = 1,
  perPage: number = 20,
): Promise<PublicRepoListResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    per_page: perPage.toString(),
  });
  return apiRequest(`/api/v1/repos/public?${params}`);
}

export async function getRepository(
  owner: string,
  repo: string,
): Promise<RepoResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
  );
}

export async function updateRepository(
  owner: string,
  repo: string,
  data: UpdateRepoRequest,
): Promise<RepoResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
    {
      method: "PATCH",
      body: JSON.stringify(data),
    },
  );
}

export async function deleteRepository(
  owner: string,
  repo: string,
): Promise<SuccessResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
    {
      method: "DELETE",
    },
  );
}

export async function getRepositoryStats(
  owner: string,
  repo: string,
): Promise<RepoStats> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/stats`,
  );
}

// ============================================================================
// Branch API
// ============================================================================

export async function listBranches(
  owner: string,
  repo: string,
): Promise<BranchListResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branches`,
  );
}

export async function createBranch(
  owner: string,
  repo: string,
  data: BranchRequest,
): Promise<{ message: string; branch: string }> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branches`,
    {
      method: "POST",
      body: JSON.stringify(data),
    },
  );
}

export async function deleteBranch(
  owner: string,
  repo: string,
  branch: string,
): Promise<SuccessResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branches/${encodeURIComponent(branch)}`,
    {
      method: "DELETE",
    },
  );
}

// ============================================================================
// Tag API
// ============================================================================

export async function listTags(
  owner: string,
  repo: string,
): Promise<TagListResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tags`,
  );
}

export async function createTag(
  owner: string,
  repo: string,
  data: TagRequest,
): Promise<{ message: string; tag: string }> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tags`,
    {
      method: "POST",
      body: JSON.stringify(data),
    },
  );
}

export async function deleteTag(
  owner: string,
  repo: string,
  tag: string,
): Promise<SuccessResponse> {
  return apiRequest(
    `/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tags/${encodeURIComponent(tag)}`,
    {
      method: "DELETE",
    },
  );
}

// ============================================================================
// Legacy API functions (for existing components - tree, blob, commits views)
// These endpoints are not in the OpenAPI spec but are used by existing components
// ============================================================================

export async function getRepos(): Promise<Repo[]> {
  try {
    const response = await listPublicRepositories(1, 100);
    return response.repositories.map((repo) => ({
      owner: repo.owner,
      name: repo.name,
      description: repo.description,
      visibility: repo.is_private ? "private" : "public",
    }));
  } catch (error) {
    console.error("Failed to fetch repos", error);
    return [];
  }
}

export async function getRepo(owner: string, name: string): Promise<Repo> {
  const repo = await getRepository(owner, name);
  return {
    owner: repo.owner,
    name: repo.name,
    description: repo.description,
    visibility: repo.is_private ? "private" : "public",
  };
}

export async function getTree(
  owner: string,
  name: string,
  urlPath: string,
): Promise<{ ref: string; path: string; entries: FileEntry[] }> {
  // Parse urlPath to extract ref and path
  // urlPath format: "ref/path/to/dir" or just "ref"
  const parts = urlPath.split("/");
  const ref = parts[0] || "HEAD";
  const path = parts.slice(1).join("/");

  const token =
    typeof window !== "undefined" ? localStorage.getItem("auth_token") : null;
  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  let url = `${API_URL}/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/tree/${encodeURIComponent(ref)}`;
  if (path) {
    url += `/${path.split("/").map(encodeURIComponent).join("/")}`;
  }

  const res = await fetch(url, { cache: "no-store", headers });
  if (!res.ok) throw new Error("Failed to fetch tree");

  const data: TreeResponse = await res.json();
  return {
    ref: data.ref,
    path: data.path,
    entries: data.entries,
  };
}

export async function getBlob(
  owner: string,
  name: string,
  urlPath: string,
): Promise<{
  ref: string;
  path: string;
  content: string;
  is_binary?: boolean;
  encoding?: string;
}> {
  // Parse urlPath to extract ref and path
  // urlPath format: "ref/path/to/file"
  const parts = urlPath.split("/");
  const ref = parts[0] || "HEAD";
  const path = parts.slice(1).join("/");

  const token =
    typeof window !== "undefined" ? localStorage.getItem("auth_token") : null;
  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const url = `${API_URL}/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/blob/${encodeURIComponent(ref)}/${path.split("/").map(encodeURIComponent).join("/")}`;

  const res = await fetch(url, { cache: "no-store", headers });
  if (!res.ok) throw new Error("Failed to fetch blob");

  const data: FileContentResponse = await res.json();
  return {
    ref: data.ref,
    path: data.path,
    content: data.content,
    is_binary: data.is_binary,
    encoding: data.encoding,
  };
}

export async function getCommits(
  owner: string,
  name: string,
  urlPath: string,
  page: number = 1,
  perPage: number = 30,
): Promise<{ ref: string; path: string; commits: Commit[] }> {
  // Parse urlPath to extract ref (and optionally path for file history)
  // urlPath format: "ref" or "ref/path/to/file"
  const parts = urlPath.split("/");
  const ref = parts[0] || "HEAD";
  const path = parts.slice(1).join("/");

  const token =
    typeof window !== "undefined" ? localStorage.getItem("auth_token") : null;
  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const params = new URLSearchParams({
    ref: ref,
    page: page.toString(),
    per_page: perPage.toString(),
  });

  const url = `${API_URL}/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/commits?${params}`;

  const res = await fetch(url, { cache: "no-store", headers });
  if (!res.ok) throw new Error("Failed to fetch commits");

  const data: CommitListResponse = await res.json();
  return {
    ref: data.ref,
    path: path,
    commits: data.commits,
  };
}

export async function getBranches(
  owner: string,
  name: string,
): Promise<Branch[]> {
  try {
    const response = await listBranches(owner, name);
    return response.branches.map((branch) => ({
      name: branch.name,
      is_head: branch.is_head,
    }));
  } catch (error) {
    // Fallback to old API if new one fails
    const url = `${API_URL}/api/v1/repos/${owner}/${name}/branches`;
    const res = await fetch(url, { cache: "no-store" });
    if (!res.ok) throw new Error("Failed to fetch branches");
    return res.json();
  }
}

export async function getDiff(
  owner: string,
  name: string,
  hash: string,
): Promise<Diff> {
  const token =
    typeof window !== "undefined" ? localStorage.getItem("auth_token") : null;
  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const url = `${API_URL}/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/diff/${encodeURIComponent(hash)}`;
  const res = await fetch(url, { cache: "no-store", headers });
  if (!res.ok) throw new Error("Failed to fetch diff");
  return res.json();
}

export async function getBlame(
  owner: string,
  name: string,
  urlPath: string,
): Promise<{ ref: string; path: string; blame: BlameLine[] }> {
  // Parse urlPath to extract ref and path
  // urlPath format: "ref/path/to/file"
  const parts = urlPath.split("/");
  const ref = parts[0] || "HEAD";
  const path = parts.slice(1).join("/");

  const token =
    typeof window !== "undefined" ? localStorage.getItem("auth_token") : null;
  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const url = `${API_URL}/api/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/blame/${encodeURIComponent(ref)}/${path.split("/").map(encodeURIComponent).join("/")}`;

  const res = await fetch(url, { cache: "no-store", headers });
  if (!res.ok) throw new Error("Failed to fetch blame");

  const data = await res.json();
  return {
    ref: data.ref,
    path: data.path,
    blame: data.blame,
  };
}
