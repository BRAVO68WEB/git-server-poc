import { getTree, getBranches } from '@/lib/api';
import { RepoFileTree } from '@/components/RepoFileTree';
import CloneCard from '@/components/clone-card';

export default async function RepoPage({
  params,
}: {
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;
  
  const httpUrl = `http://localhost:8081/${username}/${repo}.git`;
  const sshUrl = `ssh://git@localhost:2223/${username}/${repo}.git`;

  let ref = 'HEAD';
  const path = '';
  let entries: Awaited<ReturnType<typeof getTree>>['entries'] = [];
  let treeFailed = false;
  try {
    const branches = await getBranches(username, repo);
    const defaultBranch = branches.find(b => b.is_head) || branches[0];
    if (defaultBranch) {
      ref = defaultBranch.name;
    }
  } catch {
    // ignore
  }

  try {
    const data = await getTree(username, repo, ref);
    entries = data.entries;
  } catch {
    treeFailed = true;
  }

  if (treeFailed) {
    return (
      <div className="p-8 text-center border border-base rounded-lg bg-panel">
        <h3 className="text-lg font-medium text-base mb-2">Empty Repository</h3>
        <p className="text-muted">
          This repository seems to be empty or does not have a HEAD reference.
        </p>
        <div className="mt-4 p-4 bg-base rounded text-left overflow-x-auto">
            <pre className="text-sm text-base">
{`git clone https://host.xyz/${username}/${repo}
cd ${repo}
echo "# ${repo}" >> README.md
git add README.md
git commit -m "Initial commit"
git push origin HEAD`}
            </pre>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
         <div className="w-full max-w-md">
           <CloneCard httpUrl={httpUrl} sshUrl={sshUrl} />
         </div>
      </div>
      <RepoFileTree 
        owner={username} 
        name={repo} 
        currentRef={ref} 
        path={path} 
        entries={entries} 
      />
    </div>
  );
}
