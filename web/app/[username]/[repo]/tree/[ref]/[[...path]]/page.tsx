import { getTree, getBlob } from "@/lib/api";
import { RepoFileTree } from "@/components/RepoFileTree";
import Link from "next/link";

export default async function TreePage({
  params,
}: {
  params: Promise<{
    username: string;
    repo: string;
    ref: string;
    path?: string[];
  }>;
}) {
  const { username, repo, ref: refParam, path: pathSegments } = await params;

  const fullPath = [
    decodeURIComponent(refParam),
    ...(pathSegments || []).map((p) => decodeURIComponent(p)),
  ].join("/");

  let ref = "";
  let path = "";
  let entries: Awaited<ReturnType<typeof getTree>>["entries"] = [];
  let failed = false;
  let readmeContent = "";
  let readmePath = "";
  try {
    const data = await getTree(username, repo, fullPath);
    ref = data.ref;
    path = data.path;
    entries = data.entries;
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-base border border-base rounded-md bg-panel">
        Unable to load directory.
        <div className="mt-2">
          <Link
            href={`/${username}/${repo}`}
            className="text-accent hover:underline"
          >
            Back to repository
          </Link>
        </div>
      </div>
    );
  }

  const tryNames = [
    "README.md",
    "readme.md",
    "README",
    "Readme.md",
    "README.MD",
  ];
  for (const name of tryNames) {
    const candidate = path ? `${path}/${name}` : name;
    try {
      const data = await getBlob(username, repo, `${ref}/${candidate}`);
      readmeContent = data.content;
      readmePath = candidate;
      break;
    } catch {}
  }

  return (
    <div className="space-y-4">
      <RepoFileTree
        owner={username}
        name={repo}
        currentRef={ref}
        path={path}
        entries={entries}
      />
      {readmeContent && (
        <div className="border border-base rounded-md overflow-hidden bg-panel">
          <div className="px-4 py-3 border-b border-base flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm">
              <span className="font-mono bg-base px-2 py-1 rounded text-muted">
                {ref}
              </span>
              <span className="text-muted">/</span>
              <span className="font-medium text-base">{readmePath}</span>
            </div>
            <div className="flex items-center gap-2">
              <Link
                href={`/${username}/${repo}/blob/${encodeURIComponent(ref)}/${encodeURIComponent(readmePath)}`}
                className="text-xs px-2 py-1 rounded btn"
              >
                Open
              </Link>
            </div>
          </div>
          <div className="p-4 whitespace-pre-wrap font-mono text-base">
            {readmeContent}
          </div>
        </div>
      )}
    </div>
  );
}
