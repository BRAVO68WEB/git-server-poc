import { listBranches, getTree, getRepositoryStats } from "@/lib/api";
import { RepoFileTree } from "@/components/RepoFileTree";
import { env } from "@/lib/env";
import type { RepoStats } from "@/lib/types";

export default async function RepoPage({
  params,
}: {
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;

  // Fetch repository details including clone URLs
  const httpUrl = `${env.STASIS_SERVER_HOSTED_URL}/${username}/${repo}.git`;
  const sshUrl = `ssh://git@${env.STASIS_SSH_HOST_NAME}/${username}/${repo}.git`;

  let ref = "HEAD";
  const path = "";
  let entries: Awaited<ReturnType<typeof getTree>>["entries"] = [];
  let treeFailed = false;
  let stats: RepoStats | null = null;

  try {
    const [branchResponse, statsResponse] = await Promise.all([
      listBranches(username, repo),
      getRepositoryStats(username, repo).catch(() => null),
    ]);
    const branches = branchResponse.branches;
    const defaultBranch = branches.find((b) => b.is_head) || branches[0];
    if (defaultBranch) {
      ref = defaultBranch.name;
    }
    stats = statsResponse;
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
      <div className="space-y-4">
        <div className="p-8 text-center border border-base rounded-lg bg-panel">
          <h3 className="text-lg font-medium mb-2">Empty Repository</h3>
          <p className="text-muted">
            This repository seems to be empty or does not have a HEAD reference.
          </p>
          <div className="mt-4 p-4 bg-base rounded text-left overflow-x-auto">
            <pre className="text-sm">
              {`git clone ${httpUrl}
cd ${repo}
echo "# ${repo}" >> README.md
git add README.md
git commit -m "Initial commit"
git push origin HEAD

OR

git clone ${sshUrl}
cd ${repo}
echo "# ${repo}" >> README.md
git add README.md
git commit -m "Initial commit"
git push origin HEAD
`}
            </pre>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-[1fr_300px] gap-6">
      <div className="space-y-4">
        <RepoFileTree
          owner={username}
          name={repo}
          currentRef={ref}
          path={path}
          entries={entries}
        />
      </div>
      <div className="space-y-4">
        <div className="border border-base rounded-md overflow-hidden bg-panel">
          <div className="px-4 py-3 border-b border-base">
            <h3 className="font-medium text-base">Project information</h3>
          </div>
          <div className="p-4 space-y-4 text-sm">
            {stats && renderLanguageBar(stats)}
            <div className="space-y-2 pt-2">
              <div className="flex items-center gap-2">
                <span className="text-muted">⎯</span>
                <span className="font-medium">
                  {getCommitsCount(stats)} Commits
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-muted">⚭</span>
                <span className="font-medium">
                  {getBranchesCount(stats)} Branches
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-muted">🏷</span>
                <span className="font-medium">{getTagsCount(stats)} Tags</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-muted">🗃️</span>
                <span className="font-medium">{getRepoSize(stats)}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function getCommitsCount(stats: RepoStats | null): number {
  if (!stats) return 0;
  return stats.total_commits ?? stats.commits ?? 0;
}

function getBranchesCount(stats: RepoStats | null): number {
  if (!stats) return 0;
  return stats.branch_count ?? stats.branches ?? 0;
}

function getTagsCount(stats: RepoStats | null): number {
  if (!stats) return 0;
  return stats.tag_count ?? stats.tags ?? 0;
}

function getRepoSize(stats: RepoStats | null): string {
  if (!stats) return "0B";
  // Size in bytes
  let size = stats.disk_usage ?? 0;
  // Convert to human-readable format
  const units = ["B", "KB", "MB", "GB", "TB"];
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index++;
  }
  return `${size.toFixed(2)} ${units[index]}`;
}

function renderLanguageBar(stats: RepoStats) {
  const usage = stats.language_usage_perc || {};
  const entries = Object.entries(usage).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((sum, [, v]) => sum + v, 0);
  if (entries.length === 0 || total === 0) {
    return null;
  }
  return (
    <div className="space-y-2">
      <div className="h-2 w-full rounded bg-base overflow-hidden flex">
        {entries.map(([lang, pct], i) => (
          <div
            key={lang}
            style={{ width: `${pct}%`, backgroundColor: languageColor(lang, i) }}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-2 text-xs">
        {entries.map(([lang, pct], i) => (
          <div key={lang} className="flex items-center gap-1">
            <span
              className="inline-block w-2 h-2 rounded"
              style={{ backgroundColor: languageColor(lang, i) }}
            />
            <span className="text-muted">
              {lang}: {pct.toFixed(2)}%
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function languageColor(lang: string, i: number): string {
  const palette: Record<string, string> = {
    "JavaScript": "#f1e05a",
    "TypeScript": "#3178c6",
    "Go": "#00ADD8",
    "Python": "#3572A5",
    "Java": "#b07219",
    "C": "#555555",
    "C++": "#f34b7d",
    "C/C++": "#6e4c13",
    "Ruby": "#701516",
    "PHP": "#4F5D95",
    "HTML": "#e34c26",
    "CSS": "#563d7c",
    "JSON": "#292929",
    "YAML": "#cb171e",
    "Markdown": "#083fa1",
    "Rust": "#dea584",
    "Shell": "#89e051",
    "Dockerfile": "#384d54",
    "Kotlin": "#A97BFF",
    "Swift": "#F05138",
    "Dart": "#00B4AB",
    "Lua": "#000080",
    "SQL": "#e38c00",
    "XML": "#0060ac",
  };
  return palette[lang] ?? hslFromIndex(i);
}

function hslFromIndex(i: number): string {
  const hue = (i * 47) % 360;
  return `hsl(${hue}deg 60% 45%)`;
}
