import Link from 'next/link';
import { getRepos } from '@/lib/api';

export const dynamic = 'force-dynamic';

export default async function Home() {
  const repos = await getRepos();

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-3xl font-bold tracking-tight">Repositories</h1>
      </div>
      
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {repos.map((repo) => (
          <Link 
            key={`${repo.owner}/${repo.name}`} 
            href={`/${repo.owner}/${repo.name}`}
            className="group block p-6 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg hover:border-blue-500 dark:hover:border-blue-500 transition-colors"
          >
            <div className="flex items-center justify-between mb-2">
              <span className="font-semibold text-lg text-blue-600 dark:text-blue-400 group-hover:underline">
                {repo.owner} / {repo.name}
              </span>
              <span className="text-xs px-2 py-1 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700 uppercase font-medium">
                {repo.visibility}
              </span>
            </div>
            <p className="text-zinc-600 dark:text-zinc-400 text-sm line-clamp-2 h-10">
              {repo.description || 'No description provided.'}
            </p>
          </Link>
        ))}
        {repos.length === 0 && (
          <div className="col-span-full flex flex-col items-center justify-center py-20 text-center border border-dashed border-zinc-300 dark:border-zinc-700 rounded-lg">
            <p className="text-zinc-500 font-medium">No repositories found.</p>
            <p className="text-zinc-400 text-sm mt-1">Create a repository to get started.</p>
          </div>
        )}
      </div>
    </div>
  );
}
