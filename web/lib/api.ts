import {
  UserInfo,
  RegisterRequest,
  RegisterResponse,
  ChangePasswordRequest,
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
} from "./types";

const API_URL =
  process.env.NEXT_PUBLIC_API_URL ||
  process.env.API_URL ||
  "http://localhost:8080";

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
// Authentication API
// ============================================================================

export async function register(
  data: RegisterRequest,
): Promise<RegisterResponse> {
  return apiRequest("/api/v1/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getCurrentUser(): Promise<UserInfo> {
  return apiRequest("/api/v1/auth/me");
}

export async function changePassword(
  data: ChangePasswordRequest,
): Promise<SuccessResponse> {
  return apiRequest("/api/v1/auth/change-password", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

// Auth helper functions
export function logout(): void {
  removeToken();
}

export function isAuthenticated(): boolean {
  return getToken() !== null;
}

export function storeAuthToken(token: string): void {
  setToken(token);
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
    const url = `${API_URL}/api/repos/${owner}/${name}/branches`;
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
  const url = `${API_URL}/api/repos/${owner}/${name}/diff/${hash}`;
  const res = await fetch(url, { cache: "no-store" });
  if (!res.ok) throw new Error("Failed to fetch diff");
  return res.json();
}

export async function getBlame(
  owner: string,
  name: string,
  urlPath: string,
): Promise<{ ref: string; path: string; blame: BlameLine[] }> {
  const url = `${API_URL}/api/repos/${owner}/${name}/blame/${urlPath}`;
  const res = await fetch(url, { cache: "no-store" });
  if (!res.ok) throw new Error("Failed to fetch blame");
  return res.json();
}
