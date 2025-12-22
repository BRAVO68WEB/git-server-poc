import { cookies } from "next/headers";
import { UserInfo } from "./types";

// Cookie names (must match api.ts and middleware.ts)
export const AUTH_COOKIE_NAME = "auth_token";
export const USER_COOKIE_NAME = "user_info";

/**
 * Get the auth token from cookies (server-side)
 */
export async function getServerAuthToken(): Promise<string | null> {
  const cookieStore = await cookies();
  return cookieStore.get(AUTH_COOKIE_NAME)?.value || null;
}

/**
 * Get the user info from cookies (server-side)
 */
export async function getServerUserInfo(): Promise<UserInfo | null> {
  const cookieStore = await cookies();
  const userInfoStr = cookieStore.get(USER_COOKIE_NAME)?.value;
  if (!userInfoStr) return null;
  try {
    return JSON.parse(decodeURIComponent(userInfoStr)) as UserInfo;
  } catch {
    return null;
  }
}

/**
 * Check if user is authenticated (server-side)
 */
export async function isServerAuthenticated(): Promise<boolean> {
  const token = await getServerAuthToken();
  return !!token;
}

/**
 * Create headers with auth token for server-side API calls
 */
export async function getServerAuthHeaders(): Promise<HeadersInit> {
  const token = await getServerAuthToken();
  const headers: HeadersInit = {
    "Content-Type": "application/json",
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }
  return headers;
}

/**
 * Make an authenticated API request from the server
 */
export async function serverApiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  // Use NEXT_PUBLIC_API_URL if set, otherwise use default
  // For server-side in Docker, replace localhost with nginx service name
  // For client-side, http://localhost/api works fine (browser makes request to nginx)
  let API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  console.log("API_URL", API_URL);

  // If we're on the server side and running in Docker (DOCKER_ENV is set),
  // replace localhost with nginx service name
  if (
    typeof window === "undefined" &&
    API_URL.includes("localhost") &&
    process.env.DOCKER_ENV === "true"
  ) {
    API_URL = API_URL.replace("localhost", "nginx");
  }
  const authHeaders = await getServerAuthHeaders();

  const res = await fetch(`${API_URL}${endpoint}`, {
    ...options,
    headers: {
      ...authHeaders,
      ...options.headers,
    },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({
      error: "unknown_error",
      message: `Request failed with status ${res.status}`,
    }));
    throw new Error(errorData.message || "Request failed");
  }

  return res.json();
}

/**
 * Get the current user by verifying the token with the API (server-side)
 * Returns null if not authenticated or token is invalid
 */
export async function getServerCurrentUser(): Promise<UserInfo | null> {
  try {
    const token = await getServerAuthToken();
    if (!token) return null;

    const user = await serverApiRequest<UserInfo>("/api/v1/auth/me");
    return user;
  } catch {
    // Token is invalid or expired
    return null;
  }
}
