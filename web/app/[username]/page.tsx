import { listPublicRepositories, listUserRepositories } from "@/lib/api";
import { getServerCurrentUser } from "@/lib/server-auth";
import Link from "next/link";

export default async function UserReposPage({
  params,
}: {
  params: Promise<{ username: string }>;
}) {
  const { username } = await params;

  let repos: Array<{
    id: string;
    name: string;
    owner: string;
    description: string;
    is_private: boolean;
    clone_url: string;
    created_at: string;
    updated_at: string;
  }> = [];
  let failed = false;
  let isOwnProfile = false;

  // First, try to get the current user (won't throw, returns null if not authenticated)
  let currentUser = null;
  try {
    currentUser = await getServerCurrentUser();
  } catch {
    // Ignore auth errors - user is just not logged in
  }

  if (currentUser && currentUser.username === username) {
    // Viewing own profile - show all repos (public + private)
    isOwnProfile = true;
    try {
      const response = await listUserRepositories();
      repos = response.repositories || [];
    } catch (error) {
      console.error("[UserReposPage] Failed to fetch user repos:", error);
      failed = true;
    }
  } else {
    // Not logged in OR viewing another user's profile - show only their public repos
    try {
      const response = await listPublicRepositories(1, 100);
      repos = (response.repositories || []).filter((r) => r.owner === username);
    } catch (error) {
      console.error("[UserReposPage] Failed to fetch public repos:", error);
      failed = true;
    }
  }

  if (failed) {
    return (
      <div className="container mx-auto py-8 px-4">
        <div className="p-6 text-sm border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900">
          Unable to load repositories.
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8 px-4">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">
          {isOwnProfile ? (
            <>Your Repositories</>
          ) : (
            <>
              Repositories for{" "}
              <span className="text-blue-600 dark:text-blue-400">
                {username}
              </span>
            </>
          )}
        </h1>
        {isOwnProfile && (
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
        )}
      </div>

      {repos.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-center border border-dashed border-zinc-300 dark:border-zinc-700 rounded-lg">
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
          <p className="text-zinc-500 font-medium">No repositories found.</p>
          <p className="text-zinc-400 text-sm mt-1">
            {isOwnProfile
              ? "You haven't created any repositories yet."
              : `${username} hasn't created any public repositories yet.`}
          </p>
          {isOwnProfile && (
            <Link
              href="/new"
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
              Create your first repository
            </Link>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {repos.map((repo) => (
            <div
              key={repo.id}
              className="border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900 p-4 hover:border-blue-500 dark:hover:border-blue-500 transition-colors"
            >
              <div className="flex items-center gap-2">
                <Link
                  href={`/${username}/${repo.name}`}
                  className="text-blue-600 dark:text-blue-400 font-semibold hover:underline"
                >
                  {repo.name}
                </Link>
                <span
                  className={`ml-2 text-xs px-2 py-0.5 rounded-full border uppercase font-medium ${
                    repo.is_private
                      ? "bg-amber-50 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-700"
                      : "bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border-zinc-200 dark:border-zinc-700"
                  }`}
                >
                  {repo.is_private ? "private" : "public"}
                </span>
              </div>
              {repo.description && (
                <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2">
                  {repo.description}
                </p>
              )}
              <div className="mt-3 flex items-center gap-4 text-xs text-zinc-500">
                <span>
                  Updated{" "}
                  {new Date(repo.updated_at).toLocaleDateString("en-US", {
                    year: "numeric",
                    month: "short",
                    day: "numeric",
                  })}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
