"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, useEffect } from "react";
import {
  isAuthenticated,
  logout,
  logoutLocal,
  getCurrentUser,
  getStoredUserInfo,
  storeUserInfo,
} from "@/lib/api";
import { UserInfo } from "@/lib/types";

export default function Header() {
  const pathname = usePathname();
  const [user, setUser] = useState<UserInfo | null>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);

  useEffect(() => {
    const checkAuth = async () => {
      if (isAuthenticated()) {
        setIsLoggedIn(true);

        // First, try to use stored user info for immediate display
        const storedUser = getStoredUserInfo();
        if (storedUser) {
          setUser(storedUser);
        }

        // Then verify with the API and update if needed
        try {
          const userData = await getCurrentUser();
          setUser(userData);
          // Update stored user info if it changed
          storeUserInfo(userData);
        } catch {
          // Token might be invalid
          setIsLoggedIn(false);
          setUser(null);
        }
      } else {
        setIsLoggedIn(false);
        setUser(null);
      }
    };

    checkAuth();
  }, [pathname]);

  const handleLogout = async () => {
    try {
      const response = await logout();
      setIsLoggedIn(false);
      setUser(null);
      setShowUserMenu(false);

      // If the provider has a logout URL, redirect to it
      if (response?.logout_url) {
        window.location.href = response.logout_url;
      } else {
        window.location.href = "/";
      }
    } catch {
      // If logout fails, still clear local state
      logoutLocal();
      setIsLoggedIn(false);
      setUser(null);
      setShowUserMenu(false);
      window.location.href = "/";
    }
  };

  return (
    <header className="border-b border-base bg-panel">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-14">
          {/* Logo / Home Link */}
          <div className="flex items-center gap-6">
            <Link
              href="/"
              className="flex items-center gap-2 font-bold text-base hover:text-accent transition-colors"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="text-accent"
              >
                <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
                <path d="M9 18c-4.51 2-5-2-7-2" />
              </svg>
              <span>Git Server</span>
            </Link>
          </div>

          {/* Right side navigation */}
          <div className="flex items-center gap-4">
            {isLoggedIn ? (
              <>
                <Link
                  href="/new"
                  className="flex items-center gap-1 px-3 py-1.5 text-sm bg-green-600 hover:bg-green-700 text-white rounded-md transition-colors"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  New
                </Link>

                {/* User dropdown */}
                <div className="relative">
                  <button
                    onClick={() => setShowUserMenu(!showUserMenu)}
                    className="flex items-center gap-2 px-2 py-1 rounded-md hover:bg-base transition-colors"
                  >
                    <div className="w-8 h-8 rounded-full bg-accent flex items-center justify-center text-sm font-medium text-white">
                      {user?.username?.charAt(0).toUpperCase() || "U"}
                    </div>
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="text-muted"
                    >
                      <path d="m6 9 6 6 6-6" />
                    </svg>
                  </button>

                  {showUserMenu && (
                    <>
                      {/* Backdrop to close menu */}
                      <div
                        className="fixed inset-0 z-40"
                        onClick={() => setShowUserMenu(false)}
                      />

                      <div className="absolute right-0 mt-2 w-56 rounded-md shadow-lg bg-panel border border-base z-50">
                        <div className="py-1">
                          {user && (
                            <div className="px-4 py-2 border-b border-base">
                              <p className="text-sm font-medium text-base">
                                {user.username}
                              </p>
                              <p className="text-xs text-muted truncate">
                                {user.email}
                              </p>
                            </div>
                          )}

                          <Link
                            href={user ? `/${user.username}` : "/"}
                            onClick={() => setShowUserMenu(false)}
                            className="block px-4 py-2 text-sm text-base hover:bg-base transition-colors"
                          >
                            Your repositories
                          </Link>

                          <Link
                            href="/settings"
                            onClick={() => setShowUserMenu(false)}
                            className="flex items-center gap-2 px-4 py-2 text-sm text-base hover:bg-base transition-colors"
                          >
                            <svg
                              xmlns="http://www.w3.org/2000/svg"
                              width="14"
                              height="14"
                              viewBox="0 0 24 24"
                              fill="none"
                              stroke="currentColor"
                              strokeWidth="2"
                              strokeLinecap="round"
                              strokeLinejoin="round"
                            >
                              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
                              <circle cx="12" cy="12" r="3" />
                            </svg>
                            Settings
                          </Link>

                          <div className="border-t border-base my-1" />

                          <button
                            onClick={handleLogout}
                            className="w-full text-left px-4 py-2 text-sm text-red-500 hover:bg-base transition-colors"
                          >
                            Sign out
                          </button>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              </>
            ) : (
              <div className="flex items-center gap-3">
                <Link
                  href="/auth/login"
                  className="px-3 py-1.5 text-sm text-base hover:text-accent transition-colors"
                >
                  Sign in
                </Link>
                <Link
                  href="/auth/login"
                  className="px-3 py-1.5 text-sm bg-accent text-white rounded-md hover:opacity-90 transition-opacity"
                >
                  Sign up with SSO
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
