"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import {
  listPublicRepositories,
  listUserRepositories,
  isAuthenticated,
  getStoredUserInfo,
} from "@/lib/api";

interface Repository {
  id: string;
  name: string;
  owner: string;
  description: string;
  is_private: boolean;
  clone_url: string;
  created_at: string;
}

export default function Home() {
  const [repos, setRepos] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [username, setUsername] = useState<string | null>(null);

  useEffect(() => {
    const fetchRepos = async () => {
      setLoading(true);
      setError(false);

      try {
        const authenticated = isAuthenticated();
        setIsLoggedIn(authenticated);

        if (authenticated) {
          // Get user info for display
          const userInfo = getStoredUserInfo();
          if (userInfo) {
            setUsername(userInfo.username);
          }

          // Fetch user's own repositories (includes private repos)
          const response = await listUserRepositories();
          setRepos(response.repositories || []);
        } else {
          // Fetch public repositories for unauthenticated users
          const response = await listPublicRepositories(1, 50);
          setRepos(response.repositories || []);
        }
      } catch (err) {
        console.error("Failed to fetch repositories:", err);
        setError(true);
      } finally {
        setLoading(false);
      }
    };

    fetchRepos();
  }, []);

  if (loading) {
    return (
      <div className="container mx-auto py-10 px-4">
        <div className="flex items-center justify-center py-20">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-3 text-muted">Loading repositories...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {isLoggedIn ? "Your Repositories" : "Public Repositories"}
          </h1>
          {isLoggedIn && username && (
            <p className="text-muted mt-1">
              Welcome back, <span className="font-medium">{username}</span>
            </p>
          )}
        </div>
        <Link
          href="/new"
          className="inline-flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded-md shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
        >
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
          >
            <path d="M12 5v14M5 12h14" />
          </svg>
          New Repository
        </Link>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md text-sm mb-6">
          Failed to load repositories. Please try again later.
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {repos.map((repo) => (
          <Link
            key={repo.id}
            href={`/${repo.owner}/${repo.name}`}
            className="group block p-6 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg hover:border-blue-500 dark:hover:border-blue-500 transition-colors"
          >
            <div className="flex items-center justify-between mb-2">
              <span className="font-semibold text-lg text-blue-600 dark:text-blue-400 group-hover:underline">
                {repo.owner} / {repo.name}
              </span>
              <span
                className={`text-xs px-2 py-1 rounded-full border uppercase font-medium ${
                  repo.is_private
                    ? "bg-amber-50 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-700"
                    : "bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border-zinc-200 dark:border-zinc-700"
                }`}
              >
                {repo.is_private ? "private" : "public"}
              </span>
            </div>
            <p className="text-zinc-600 dark:text-zinc-400 text-sm line-clamp-2 h-10">
              {repo.description || "No description provided."}
            </p>
            <div className="mt-3 text-xs text-zinc-500">
              Created{" "}
              {new Date(repo.created_at).toLocaleDateString("en-US", {
                year: "numeric",
                month: "short",
                day: "numeric",
              })}
            </div>
          </Link>
        ))}
        {repos.length === 0 && !error && (
          <div className="col-span-full flex flex-col items-center justify-center py-20 text-center border border-dashed border-zinc-300 dark:border-zinc-700 rounded-lg">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="48"
              height="48"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="text-zinc-400 mb-4"
            >
              <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
              <path d="M9 18c-4.51 2-5-2-7-2" />
            </svg>
            <p className="text-zinc-500 font-medium">
              {isLoggedIn
                ? "You don't have any repositories yet."
                : "No public repositories found."}
            </p>
            <p className="text-zinc-400 text-sm mt-1">
              {isLoggedIn
                ? "Create a repository to get started."
                : "Sign in to see your repositories or create a new one."}
            </p>
            <Link
              href={isLoggedIn ? "/new" : "/auth/login"}
              className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded-md shadow-sm transition-colors"
            >
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
              >
                <path d="M12 5v14M5 12h14" />
              </svg>
              {isLoggedIn ? "Create your first repository" : "Sign in"}
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
