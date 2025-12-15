import Link from 'next/link';
import { getRepo, getBranches } from '@/lib/api';
import BranchSelector from '@/components/branch-selector';

export default async function RepoLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;
  
  let repoData: { visibility?: string; description?: string } = {};
  let branches: Array<{ name: string; is_head: boolean }> = [];
  try {
    repoData = await getRepo(username, repo);
  } catch {
    repoData = {};
  }
  try {
    branches = await getBranches(username, repo);
  } catch {
    branches = [];
  }

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="mb-6">
        <div className="flex items-center gap-2 text-xl mb-2">
          <span className="text-accent">
            {/* User profile link could go here */}
            <span className="hover:underline cursor-pointer">{username}</span>
          </span>
          <span className="text-muted">/</span>
          <Link href={`/${username}/${repo}`} className="font-bold text-accent hover:underline">
            {repo}
          </Link>
          <span className="ml-2 text-xs px-2 py-0.5 rounded-full border border-base text-muted uppercase font-medium">
            {repoData.visibility || ''}
          </span>
        </div>
        <p className="text-muted">{repoData.description || ''}</p>
      </div>

      <div className="border-b border-base mb-6 flex justify-between items-center">
        <nav className="flex gap-6 -mb-px">
          <Link 
            href={`/${username}/${repo}`} 
            className="border-b-2 border-base font-medium text-base pb-3 px-1"
          >
            Code
          </Link>
          <Link 
            href={`/${username}/${repo}/commits`}
            className="text-muted pb-3 px-1 hover:text-accent"
          >
            Commits
          </Link>
        </nav>
        <div className="mb-2 flex items-center gap-2 text-muted bg-panel px-2 py-1 rounded-md border border-base">
           <BranchSelector branches={branches} />
        </div>
      </div>

      {children}
    </div>
  );
}
