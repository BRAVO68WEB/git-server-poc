import Link from "next/link";
import { getRepository, listBranches, listTags } from "@/lib/api";
import BranchSelector from "@/components/branch-selector";

export default async function RepoLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;

  let repoData: {
    is_private?: boolean;
    description?: string;
  } = {};
  let branches: Array<{ name: string; hash: string; is_head: boolean }> = [];
  let branchCount = 0;
  let tagCount = 0;

  try {
    const repoResponse = await getRepository(username, repo);
    repoData = {
      is_private: repoResponse.is_private,
      description: repoResponse.description,
    };
  } catch {
    repoData = {};
  }

  try {
    const branchResponse = await listBranches(username, repo);
    branches = branchResponse.branches;
    branchCount = branchResponse.total;
  } catch {
    branches = [];
  }

  try {
    const tagResponse = await listTags(username, repo);
    tagCount = tagResponse.total;
  } catch {
    tagCount = 0;
  }

  const visibility = repoData.is_private ? "private" : "public";

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="mb-6">
        <div className="flex items-center gap-2 text-xl mb-2">
          <span className="text-accent">
            <Link
              href={`/${username}`}
              className="hover:underline cursor-pointer"
            >
              {username}
            </Link>
          </span>
          <span className="text-muted">/</span>
          <Link
            href={`/${username}/${repo}`}
            className="font-bold text-accent hover:underline"
          >
            {repo}
          </Link>
          <span className="ml-2 text-xs px-2 py-0.5 rounded-full border border-base text-muted uppercase font-medium">
            {visibility}
          </span>
        </div>
        <p className="text-muted">{repoData.description || ""}</p>
      </div>

      <div className="border-b border-base mb-6 flex justify-between items-center">
        <nav className="flex gap-6 -mb-px">
          <Link
            href={`/${username}/${repo}`}
            className="border-b-2 border-transparent hover:border-base font-medium text-base pb-3 px-1 hover:text-accent"
          >
            Code
          </Link>
          <Link
            href={`/${username}/${repo}/commits`}
            className="border-b-2 border-transparent hover:border-base text-muted pb-3 px-1 hover:text-accent"
          >
            Commits
          </Link>
          <Link
            href={`/${username}/${repo}/branches`}
            className="border-b-2 border-transparent hover:border-base text-muted pb-3 px-1 hover:text-accent"
          >
            Branches
            {branchCount > 0 && (
              <span className="ml-1 text-xs bg-base px-1.5 py-0.5 rounded-full">
                {branchCount}
              </span>
            )}
          </Link>
          <Link
            href={`/${username}/${repo}/tags`}
            className="border-b-2 border-transparent hover:border-base text-muted pb-3 px-1 hover:text-accent"
          >
            Tags
            {tagCount > 0 && (
              <span className="ml-1 text-xs bg-base px-1.5 py-0.5 rounded-full">
                {tagCount}
              </span>
            )}
          </Link>
          <Link
            href={`/${username}/${repo}/settings`}
            className="border-b-2 border-transparent hover:border-base text-muted pb-3 px-1 hover:text-accent"
          >
            Settings
          </Link>
        </nav>
        <div className="mb-2 flex items-center gap-2 text-muted bg-panel px-2 py-1 rounded-md border border-base">
          <BranchSelector
            branches={branches.map((b) => ({
              name: b.name,
              is_head: b.is_head,
            }))}
          />
        </div>
      </div>

      {children}
    </div>
  );
}
