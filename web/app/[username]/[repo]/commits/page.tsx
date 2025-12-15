import { getBranches, getCommits } from '@/lib/api';
import Link from 'next/link';

export default async function CommitsIndexPage({
  params,
}: {
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;
  let ref = 'HEAD';
  let commits: Array<{ hash: string; author: string; date: string; message: string }> = [];
  let failed = false;

  try {
    const branches = await getBranches(username, repo);
    const defaultBranch = branches.find(b => b.is_head) || branches[0];
    if (defaultBranch) {
      ref = defaultBranch.name;
    }
    const data = await getCommits(username, repo, ref);
    commits = data.commits || [];
    ref = data.ref;
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
        Unable to load commits.
        <div className="mt-2">
          <Link href={`/${username}/${repo}`} className="text-accent hover:underline">
            Back to repository
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-base rounded-md overflow-hidden bg-panel">
      <div className="px-4 py-3 border-b border-base flex items-center gap-2 text-sm">
        <span className="font-semibold text-base">Commits</span>
        <span className="bg-base px-2 py-0.5 rounded text-muted text-xs">
          {ref}
        </span>
      </div>
      <div className="divide-y divide-[var(--border-base)]">
        {commits.map((commit) => (
          <div key={commit.hash} className="p-4 hover:bg-base transition-colors flex items-start gap-4">
            <div className="flex-1 min-w-0">
              <p className="font-semibold text-base truncate">
                {commit.message}
              </p>
              <div className="flex items-center gap-2 mt-1 text-xs text-muted">
                <span className="font-medium text-base">{commit.author}</span>
                <span>committed on {new Date(commit.date).toLocaleDateString()}</span>
              </div>
            </div>
            <div className="flex items-center">
              <div className="flex border border-base rounded-md overflow-hidden text-xs font-mono">
                <span className="bg-base px-2 py-1 text-muted border-r border-base">
                  commit
                </span>
                <Link
                  href={`/${username}/${repo}/commit/${commit.hash}`}
                  className="px-2 py-1 text-accent bg-panel hover:underline"
                >
                  {commit.hash.substring(0, 7)}
                </Link>
              </div>
              <Link
                href={`/${username}/${repo}/tree/${commit.hash}`}
                className="ml-2 text-xs px-2 py-1 rounded btn"
              >
                Browse Files
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
