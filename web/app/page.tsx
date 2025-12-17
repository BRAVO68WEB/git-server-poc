import Link from "next/link";
import { listPublicRepositories } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function Home() {
  let repos: Array<{
    id: string;
    name: string;
    owner: string;
    description: string;
    is_private: boolean;
    clone_url: string;
    created_at: string;
  }> = [];
  let error = false;

  try {
    const response = await listPublicRepositories(1, 50);
    repos = response.repositories;
  } catch {
    error = true;
  }

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-3xl font-bold tracking-tight">Repositories</h1>
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
              <span className="text-xs px-2 py-1 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700 uppercase font-medium">
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
            <p className="text-zinc-500 font-medium">No repositories found.</p>
            <p className="text-zinc-400 text-sm mt-1">
              Create a repository to get started.
            </p>
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
          </div>
        )}
      </div>
    </div>
  );
}
