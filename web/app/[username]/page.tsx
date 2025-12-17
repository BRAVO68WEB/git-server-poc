import { listPublicRepositories } from "@/lib/api";
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

  try {
    // Fetch public repositories and filter by username
    const response = await listPublicRepositories(1, 100);
    repos = response.repositories.filter((r) => r.owner === username);
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="container mx-auto py-8 px-4">
        <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
          Unable to load repositories.
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8 px-4">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-base">
          Repositories for <span className="text-accent">{username}</span>
        </h1>
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
            {username} hasn&apos;t created any public repositories yet.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {repos.map((repo) => (
            <div
              key={repo.id}
              className="border border-base rounded-md bg-panel p-4 hover:border-blue-500 dark:hover:border-blue-500 transition-colors"
            >
              <div className="flex items-center gap-2">
                <Link
                  href={`/${username}/${repo.name}`}
                  className="text-accent font-semibold hover:underline"
                >
                  {repo.name}
                </Link>
                <span className="ml-2 text-xs px-2 py-0.5 rounded-full border border-base text-muted uppercase font-medium">
                  {repo.is_private ? "private" : "public"}
                </span>
              </div>
              {repo.description && (
                <p className="mt-2 text-sm text-muted line-clamp-2">
                  {repo.description}
                </p>
              )}
              <div className="mt-3 flex items-center gap-4 text-xs text-muted">
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
