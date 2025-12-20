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
  const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
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
