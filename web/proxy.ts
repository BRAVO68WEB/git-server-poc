import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Cookie names (must match api.ts)
const AUTH_COOKIE_NAME = "auth_token";

// Routes that require authentication
const PROTECTED_ROUTES = [
  "/settings",
  "/new",
];

// Routes that should redirect to home if already authenticated
const AUTH_ROUTES = [
  "/auth/login",
  "/auth/register",
];

// Public routes that don't need any auth check
const PUBLIC_ROUTES = [
  "/",
  "/auth/callback",
  "/_next",
  "/api",
  "/favicon.ico",
];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  
  // Skip middleware for static files and API routes
  if (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.includes(".") // Static files like favicon.ico
  ) {
    return NextResponse.next();
  }

  // Get auth token from cookie
  const authToken = request.cookies.get(AUTH_COOKIE_NAME)?.value;
  const isAuthenticated = !!authToken;

  // Check if the route requires authentication
  const isProtectedRoute = PROTECTED_ROUTES.some((route) =>
    pathname.startsWith(route)
  );

  // Check if the route is an auth route (login/register)
  const isAuthRoute = AUTH_ROUTES.some((route) =>
    pathname.startsWith(route)
  );

  // Redirect to login if accessing protected route without auth
  if (isProtectedRoute && !isAuthenticated) {
    const loginUrl = new URL("/auth/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Redirect to home if accessing auth routes while already authenticated
  if (isAuthRoute && isAuthenticated) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  // For authenticated requests, forward the auth token in headers for SSR API calls
  if (isAuthenticated) {
    const response = NextResponse.next();
    // Add auth token to request headers for server components
    const requestHeaders = new Headers(request.headers);
    requestHeaders.set("x-auth-token", authToken);
    
    return NextResponse.next({
      request: {
        headers: requestHeaders,
      },
    });
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    "/((?!_next/static|_next/image|favicon.ico|public/).*)",
  ],
};
