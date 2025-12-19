"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { getOIDCConfig, initiateOIDCLogin, isAuthenticated } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [oidcInitialized, setOidcInitialized] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Check if already authenticated
    if (isAuthenticated()) {
      router.push("/");
      return;
    }

    // Check OIDC configuration
    const checkOIDC = async () => {
      try {
        const config = await getOIDCConfig();
        setOidcEnabled(config.oidc_enabled);
        setOidcInitialized(config.oidc_initialized);
      } catch (err) {
        console.error("Failed to check OIDC config:", err);
        setError("Failed to load authentication configuration");
      } finally {
        setLoading(false);
      }
    };

    checkOIDC();
  }, [router]);

  const handleLogin = () => {
    setLoading(true);
    initiateOIDCLogin();
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center px-4">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-muted">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-bold text-base">
            Sign in to your account
          </h2>
          <p className="mt-2 text-center text-sm text-muted">
            Use your organization&apos;s identity provider to sign in
          </p>
        </div>

        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md text-sm">
            {error}
          </div>
        )}

        {!oidcEnabled ? (
          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 text-yellow-600 dark:text-yellow-400 px-4 py-3 rounded-md text-sm">
            <p className="font-medium">OIDC Authentication Not Configured</p>
            <p className="mt-1">
              Please contact your administrator to configure OIDC
              authentication.
            </p>
          </div>
        ) : !oidcInitialized ? (
          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 text-yellow-600 dark:text-yellow-400 px-4 py-3 rounded-md text-sm">
            <p className="font-medium">OIDC Service Unavailable</p>
            <p className="mt-1">
              The authentication service is temporarily unavailable. Please try
              again later.
            </p>
          </div>
        ) : (
          <div className="mt-8 space-y-6">
            <button
              onClick={handleLogin}
              disabled={loading}
              className="w-full flex justify-center items-center gap-2 py-3 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"
                />
              </svg>
              Sign in with SSO
            </button>

            <p className="text-center text-xs text-muted">
              You will be redirected to your identity provider to authenticate.
            </p>
          </div>
        )}

        <div className="mt-6">
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-base" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-base text-muted">Using SSH keys?</span>
            </div>
          </div>

          <div className="mt-6 text-center">
            <p className="text-sm text-muted">
              After signing in, you can add SSH keys for passwordless Git
              access.{" "}
              <Link
                href="/settings/ssh-keys"
                className="text-accent hover:underline"
              >
                Learn more
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
