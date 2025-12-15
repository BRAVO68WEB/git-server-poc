import { getRepos } from '@/lib/api';
import Link from 'next/link';

export default async function UserReposPage({
  params,
}: {
  params: Promise<{ username: string }>;
}) {
  const { username } = await params;

  let repos: Array<{ owner: string; name: string; description?: string; visibility?: string }> = [];
  let failed = false;
  try {
    const all = await getRepos();
    repos = all.filter(r => r.owner === username);
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
        Unable to load repositories.
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8 px-4">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-base">
          Repositories for <span className="text-accent">{username}</span>
        </h1>
      </div>
      {repos.length === 0 ? (
        <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
          No repositories found.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {repos.map((repo) => (
            <div key={repo.name} className="border border-base rounded-md bg-panel p-4">
              <div className="flex items-center gap-2">
                <Link href={`/${username}/${repo.name}`} className="text-accent font-semibold hover:underline">
                  {repo.name}
                </Link>
                <span className="ml-2 text-xs px-2 py-0.5 rounded-full border border-base text-muted uppercase font-medium">
                  {repo.visibility || ''}
                </span>
              </div>
              {repo.description && (
                <p className="mt-2 text-sm text-muted">{repo.description}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
