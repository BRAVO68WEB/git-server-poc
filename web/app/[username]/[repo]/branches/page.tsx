import { getBranches } from '@/lib/api';
import Link from 'next/link';

export default async function BranchesPage({
  params,
}: {
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;
  let branches: Array<{ name: string; is_head: boolean }> = [];
  let failed = false;

  try {
    branches = await getBranches(username, repo);
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
        Unable to load branches.
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
      <div className="px-4 py-3 border-b border-base">
        <span className="font-semibold text-base">Branches</span>
      </div>
      <div className="divide-y divide-[var(--border-base)]">
        {branches.map((b) => (
          <div key={b.name} className="p-4 flex items-center justify-between hover:bg-base">
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm text-base">{b.name}</span>
              {b.is_head && (
                <span className="text-xs px-2 py-0.5 rounded border border-base text-muted">
                  default
                </span>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Link href={`/${username}/${repo}/tree/${b.name}`} className="text-xs px-2 py-1 rounded btn">
                Browse
              </Link>
              <Link href={`/${username}/${repo}/commits/${b.name}`} className="text-xs px-2 py-1 rounded btn">
                Commits
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
