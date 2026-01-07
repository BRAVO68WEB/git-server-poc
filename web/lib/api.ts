import {
  UserInfo,
  UpdateUserResponse,
  OIDCConfigResponse,
  OIDCCallbackResponse,
  OIDCLogoutResponse,
  CreateRepoRequest,
  ImportRepoRequest,
  UpdateRepoRequest,
  UpdateMirrorSettingsRequest,
  MirrorSettingsResponse,
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
  TokenInfo,
  CreateTokenRequest,
  CreateTokenResponse,
  ListTokensResponse,
  CIArtifact,
} from "./types";
import { env } from "./env";

// Get API URL - handle server vs client differently
function getApiUrl(): string {
  const baseUrl = env.NEXT_PUBLIC_API_URL;

  // If we're on the server side (SSR) and running in Docker (DOCKER_ENV is set),
  // replace localhost with nginx service name
  if (
    typeof window === "undefined" &&
    env.DOCKER_ENV === "true" &&
    baseUrl.includes("localhost")
  ) {
    return baseUrl.replace("localhost", "nginx");
  }

  return baseUrl;
}

// Cookie names
export const AUTH_COOKIE_NAME = "auth_token";
export const USER_COOKIE_NAME = "user_info";

// Cookie helpers for client-side
function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) {
    const cookieValue = parts.pop()?.split(";").shift();
    return cookieValue ? decodeURIComponent(cookieValue) : null;
  }
  return null;
}

function setCookie(name: string, value: string, days: number = 7): void {
  if (typeof document === "undefined") return;
  const expires = new Date();
  expires.setTime(expires.getTime() + days * 24 * 60 * 60 * 1000);
  // Set cookie with SameSite=Lax for security, path=/ for all routes
  document.cookie = `${name}=${encodeURIComponent(value)};expires=${expires.toUTCString()};path=/;SameSite=Lax`;
}

function deleteCookie(name: string): void {
  if (typeof document === "undefined") return;
  document.cookie = `${name}=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/;SameSite=Lax`;
}

// Token helpers using cookies
function getToken(): string | null {
  return getCookie(AUTH_COOKIE_NAME);
}

// Get token for server-side (async, uses next/headers)
async function getServerToken(): Promise<string | null> {
  try {
    const { cookies } = await import("next/headers");
    const cookieStore = await cookies();
    return cookieStore.get(AUTH_COOKIE_NAME)?.value || null;
  } catch {
    return null;
  }
}

function setToken(token: string): void {
  setCookie(AUTH_COOKIE_NAME, token, 7); // 7 days expiry
}

function removeToken(): void {
  deleteCookie(AUTH_COOKIE_NAME);
}

// User info helpers using cookies
export function getUserInfo(): UserInfo | null {
  const userInfoStr = getCookie(USER_COOKIE_NAME);
  if (!userInfoStr) return null;
  try {
    return JSON.parse(userInfoStr) as UserInfo;
  } catch {
    return null;
  }
}

export function setUserInfo(user: UserInfo): void {
  setCookie(USER_COOKIE_NAME, JSON.stringify(user), 7);
}

export function removeUserInfo(): void {
  deleteCookie(USER_COOKIE_NAME);
}

// Check if running on server
function isServer(): boolean {
  return typeof window === "undefined";
}

// Request helper with authentication (works on both client and server)
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  // Get token - use server method on server, client method on client
  let token: string | null = null;
  if (isServer()) {
    token = await getServerToken();
  } else {
    token = getToken();
  }

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  if (token) {
    // All tokens are now Bearer tokens (JWT from OIDC or PAT)
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${getApiUrl()}${endpoint}`, {
    ...options,
    headers,
    credentials: "include", // Include cookies for cross-origin requests
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
  return apiRequest("/v1/auth/oidc/config");
}

/**
 * Get the OIDC login URL
 * The user should be redirected to this URL to start the OIDC flow
 */
export function getOIDCLoginURL(): string {
  return `${getApiUrl()}/v1/auth/oidc/login`;
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
 * The token is automatically stored in cookies
 */
export async function handleOIDCCallback(
  code: string,
  state: string,
): Promise<OIDCCallbackResponse> {
  // The callback is handled server-side, but we need to process the response
  // This function is called after the redirect, when we have the token in the response
  const response = await apiRequest<OIDCCallbackResponse>(
    `/v1/auth/oidc/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
  );

  // Store the session token in cookie
  if (response.token) {
    setToken(response.token);
  }

  // Store the user info in cookie
  if (response.user) {
    setUserInfo(response.user);
  }

  return response;
}

/**
 * Store the auth token in cookie (called from callback page)
 */
export function storeAuthToken(token: string): void {
  setToken(token);
}

/**
 * Store user info in cookie
 */
export function storeUserInfo(user: UserInfo): void {
  setUserInfo(user);
}

/**
 * Get the current authenticated user
 */
export async function getCurrentUser(): Promise<UserInfo> {
  return apiRequest("/v1/auth/me");
}

/**
 * Update current user's username
 */
export async function updateUsername(
  username: string,
): Promise<UpdateUserResponse> {
  return apiRequest<UpdateUserResponse>("/v1/users/username", {
    method: "PUT",
    body: JSON.stringify({ username }),
  });
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
      `/v1/auth/oidc/logout${params}`,
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
 * Logout locally only (clear cookies without calling the API)
 */
export function logoutLocal(): void {
  removeToken();
  removeUserInfo();
}

/**
 * Get stored user info from cookie (without API call)
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
  return apiRequest("/v1/ssh-keys");
}

export async function addSSHKey(
  data: AddSSHKeyRequest,
): Promise<AddSSHKeyResponse> {
  return apiRequest("/v1/ssh-keys", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getSSHKey(id: string): Promise<SSHKeyInfo> {
  return apiRequest(`/v1/ssh-keys/${encodeURIComponent(id)}`);
}

export async function deleteSSHKey(id: string): Promise<SuccessResponse> {
  return apiRequest(`/v1/ssh-keys/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

// ============================================================================
// Personal Access Token (PAT) API
// ============================================================================

export async function listTokens(): Promise<ListTokensResponse> {
  return apiRequest("/v1/tokens");
}

export async function createToken(
  data: CreateTokenRequest,
): Promise<CreateTokenResponse> {
  return apiRequest("/v1/tokens", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteToken(id: string): Promise<SuccessResponse> {
  return apiRequest(`/v1/tokens/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

// ============================================================================
// Repository API
// ============================================================================

export async function listUserRepositories(): Promise<RepoListResponse> {
  return apiRequest("/v1/repos");
}

export async function createRepository(
  data: CreateRepoRequest,
): Promise<RepoResponse> {
  return apiRequest("/v1/repos", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function importRepository(
  data: ImportRepoRequest,
): Promise<RepoResponse> {
  return apiRequest("/v1/repos/import", {
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
  return apiRequest(`/v1/repos/public?${params}`);
}

export async function getRepository(
  owner: string,
  repo: string,
): Promise<RepoResponse> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
  );
}

export async function updateRepository(
  owner: string,
  repo: string,
  data: UpdateRepoRequest,
): Promise<RepoResponse> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
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
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
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
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/stats`,
  );
}

export async function getMirrorSettings(
  owner: string,
  repo: string,
): Promise<MirrorSettingsResponse> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/mirror`,
  );
}

export async function updateMirrorSettings(
  owner: string,
  repo: string,
  data: UpdateMirrorSettingsRequest,
): Promise<RepoResponse> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/mirror`,
    {
      method: "PATCH",
      body: JSON.stringify(data),
    },
  );
}

export async function syncMirrorRepository(
  owner: string,
  repo: string,
): Promise<{ message: string; status: string }> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/sync`,
    {
      method: "POST",
    },
  );
}

export async function getMirrorStatus(
  owner: string,
  repo: string,
): Promise<MirrorSettingsResponse> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/mirror/status`,
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
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branches`,
  );
}

export async function createBranch(
  owner: string,
  repo: string,
  data: BranchRequest,
): Promise<{ message: string; branch: string }> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branches`,
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
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branches/${encodeURIComponent(branch)}`,
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
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tags`,
  );
}

export async function createTag(
  owner: string,
  repo: string,
  data: TagRequest,
): Promise<{ message: string; tag: string }> {
  return apiRequest(
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tags`,
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
    `/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tags/${encodeURIComponent(tag)}`,
    {
      method: "DELETE",
    },
  );
}

// ============================================================================
// Legacy API functions (for existing components - tree, blob, commits views)
// These endpoints are not in the OpenAPI spec but are used by existing components
// ============================================================================

// Helper to get auth headers for legacy functions (async, works on both client and server)
async function getLegacyAuthHeaders(): Promise<HeadersInit> {
  let token: string | null = null;
  if (isServer()) {
    token = await getServerToken();
  } else {
    token = getToken();
  }

  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }
  return headers;
}

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

  const headers = await getLegacyAuthHeaders();

  let url = `${getApiUrl()}/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/tree/${encodeURIComponent(ref)}`;
  if (path) {
    url += `/${path.split("/").map(encodeURIComponent).join("/")}`;
  }

  const res = await fetch(url, {
    cache: "no-store",
    headers,
    credentials: "include",
  });
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

  const headers = await getLegacyAuthHeaders();

  const url = `${getApiUrl()}/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/blob/${encodeURIComponent(ref)}/${path.split("/").map(encodeURIComponent).join("/")}`;

  const res = await fetch(url, {
    cache: "no-store",
    headers,
    credentials: "include",
  });
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

  const headers = await getLegacyAuthHeaders();

  const params = new URLSearchParams({
    ref: ref,
    page: page.toString(),
    per_page: perPage.toString(),
  });

  const url = `${getApiUrl()}/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/commits?${params}`;

  const res = await fetch(url, {
    cache: "no-store",
    headers,
    credentials: "include",
  });
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
    const headers = await getLegacyAuthHeaders();
    const url = `${getApiUrl()}/v1/repos/${owner}/${name}/branches`;
    const res = await fetch(url, {
      cache: "no-store",
      headers,
      credentials: "include",
    });
    if (!res.ok) throw new Error("Failed to fetch branches");
    return res.json();
  }
}

export async function getDiff(
  owner: string,
  name: string,
  hash: string,
): Promise<Diff> {
  const headers = await getLegacyAuthHeaders();

  const url = `${getApiUrl()}/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/diff/${encodeURIComponent(hash)}`;
  const res = await fetch(url, {
    cache: "no-store",
    headers,
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to fetch diff");
  return res.json();
}

export async function getCompareDiff(
  owner: string,
  name: string,
  from: string,
  to: string,
): Promise<Diff> {
  const headers = await getLegacyAuthHeaders();
  const range = `${encodeURIComponent(from)}..${encodeURIComponent(to)}`;
  const url = `${getApiUrl()}/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/compare/${range}`;
  const res = await fetch(url, {
    cache: "no-store",
    headers,
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to fetch compare diff");
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

  const headers = await getLegacyAuthHeaders();

  const url = `${getApiUrl()}/v1/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/blame/${encodeURIComponent(ref)}/${path.split("/").map(encodeURIComponent).join("/")}`;

  const res = await fetch(url, {
    cache: "no-store",
    headers,
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to fetch blame");

  const data = await res.json();
  return {
    ref: data.ref,
    path: data.path,
    blame: data.blame,
  };
}

// ============================================================================
// CI/CD API
// ============================================================================

import {
  CIJob,
  CIJobListResponse,
  CIJobLogsResponse,
  TriggerCIJobRequest,
  TriggerCIJobResponse,
  CIJobEvent,
  CIJobLog,
} from "./types";

/**
 * List CI jobs for a repository
 */
export async function listCIJobs(
  owner: string,
  repo: string,
  limit: number = 20,
  offset: number = 0,
): Promise<CIJobListResponse> {
  return apiRequest<CIJobListResponse>(
    `/v1/repos/${owner}/${repo}/ci/jobs?limit=${limit}&offset=${offset}`,
  );
}

/**
 * Get a specific CI job
 */
export async function getCIJob(
  owner: string,
  repo: string,
  jobId: string,
): Promise<CIJob> {
  return apiRequest<CIJob>(`/v1/repos/${owner}/${repo}/ci/jobs/${jobId}`);
}

/**
 * Get the latest CI job for a repository
 */
export async function getLatestCIJob(
  owner: string,
  repo: string,
): Promise<CIJob> {
  return apiRequest<CIJob>(`/v1/repos/${owner}/${repo}/ci/latest`);
}

/**
 * Get CI job logs
 */
export async function getCIJobLogs(
  owner: string,
  repo: string,
  jobId: string,
  limit: number = 1000,
  offset: number = 0,
): Promise<CIJobLogsResponse> {
  return apiRequest<CIJobLogsResponse>(
    `/v1/repos/${owner}/${repo}/ci/jobs/${jobId}/logs?limit=${limit}&offset=${offset}`,
  );
}

/**
 * Trigger a new CI job
 */
export async function triggerCIJob(
  owner: string,
  repo: string,
  request: TriggerCIJobRequest,
): Promise<TriggerCIJobResponse> {
  return apiRequest<TriggerCIJobResponse>(
    `/v1/repos/${owner}/${repo}/ci/jobs`,
    {
      method: "POST",
      body: JSON.stringify(request),
    },
  );
}

/**
 * Cancel a running CI job
 */
export async function cancelCIJob(
  owner: string,
  repo: string,
  jobId: string,
): Promise<{ message: string; job_id: string }> {
  return apiRequest<{ message: string; job_id: string }>(
    `/v1/repos/${owner}/${repo}/ci/jobs/${jobId}/cancel`,
    { method: "POST" },
  );
}

/**
 * Retry a failed CI job
 */
export async function retryCIJob(
  owner: string,
  repo: string,
  jobId: string,
): Promise<{
  message: string;
  new_job_id: string;
  new_run_id: string;
  original_job_id: string;
}> {
  return apiRequest<{
    message: string;
    new_job_id: string;
    new_run_id: string;
    original_job_id: string;
  }>(`/v1/repos/${owner}/${repo}/ci/jobs/${jobId}/retry`, { method: "POST" });
}

/**
 * List artifacts for a CI job
 */
export async function listCIJobArtifacts(
  owner: string,
  repo: string,
  jobId: string,
): Promise<{ artifacts: CIArtifact[]; total: number }> {
  return apiRequest<{ artifacts: CIArtifact[]; total: number }>(
    `/v1/repos/${owner}/${repo}/ci/jobs/${jobId}/artifacts`,
  );
}

/**
 * Get artifact download URL
 */
export function getArtifactDownloadUrl(
  owner: string,
  repo: string,
  jobId: string,
  artifactName: string,
): string {
  return `${getApiUrl()}/v1/repos/${owner}/${repo}/ci/jobs/${jobId}/artifacts/${encodeURIComponent(artifactName)}`;
}

/**
 * Subscribe to CI job log stream via Server-Sent Events
 * Returns an EventSource that emits job events in real-time
 */
export function subscribeToCIJobStream(
  owner: string,
  repo: string,
  jobId: string,
  onEvent: (event: CIJobEvent) => void,
  onError?: (error: Event) => void,
): EventSource {
  const url = `${getApiUrl()}/v1/repos/${owner}/${repo}/ci/jobs/${jobId}/stream`;
  const eventSource = new EventSource(url, { withCredentials: true });

  // Handle connection established
  eventSource.addEventListener("connected", (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data);
      onEvent({ type: "connected", job_id: jobId, data });
    } catch (err) {
      console.error("Failed to parse connected event:", err);
    }
  });

  // Handle status updates
  eventSource.addEventListener("status", (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data);
      onEvent({ type: "status", job_id: jobId, data });
    } catch (err) {
      console.error("Failed to parse status event:", err);
    }
  });

  // Handle log entries
  eventSource.addEventListener("log", (e: MessageEvent) => {
    try {
      const data: CIJobLog = JSON.parse(e.data);
      onEvent({ type: "log", job_id: jobId, data });
    } catch (err) {
      console.error("Failed to parse log event:", err);
    }
  });

  // Handle step updates
  eventSource.addEventListener("step", (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data);
      onEvent({ type: "step", job_id: jobId, data });
    } catch (err) {
      console.error("Failed to parse step event:", err);
    }
  });

  // Handle artifact updates
  eventSource.addEventListener("artifact", (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data);
      onEvent({ type: "artifact", job_id: jobId, data });
    } catch (err) {
      console.error("Failed to parse artifact event:", err);
    }
  });

  // Handle errors
  eventSource.onerror = (e) => {
    if (onError) {
      onError(e);
    }
  };

  return eventSource;
}

/**
 * Get status badge color for CI job status
 */
export function getCIStatusColor(status: string): string {
  switch (status) {
    case "success":
      return "text-green-500";
    case "failed":
    case "error":
      return "text-red-500";
    case "running":
      return "text-blue-500";
    case "pending":
    case "queued":
      return "text-yellow-500";
    case "cancelled":
      return "text-gray-500";
    case "timed_out":
      return "text-orange-500";
    default:
      return "text-muted";
  }
}

/**
 * Get status badge background color for CI job status
 */
export function getCIStatusBgColor(status: string): string {
  switch (status) {
    case "success":
      return "bg-green-500/10 border-green-500/30";
    case "failed":
    case "error":
      return "bg-red-500/10 border-red-500/30";
    case "running":
      return "bg-blue-500/10 border-blue-500/30";
    case "pending":
    case "queued":
      return "bg-yellow-500/10 border-yellow-500/30";
    case "cancelled":
      return "bg-gray-500/10 border-gray-500/30";
    case "timed_out":
      return "bg-orange-500/10 border-orange-500/30";
    default:
      return "bg-base border-base";
  }
}

/**
 * Format duration in seconds to human-readable string
 */
export function formatCIDuration(seconds: number | undefined): string {
  if (seconds === undefined || seconds === null) {
    return "-";
  }

  if (seconds < 60) {
    return `${Math.round(seconds)}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.round(seconds % 60);

  if (minutes < 60) {
    return `${minutes}m ${remainingSeconds}s`;
  }

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}
