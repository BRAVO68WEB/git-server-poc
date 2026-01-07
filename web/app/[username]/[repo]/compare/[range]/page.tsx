import { getCompareDiff } from "@/lib/api";
import Link from "next/link";
import type { Diff } from "@/lib/types";
import { DiffViewer } from "@/components/DiffViewer";

export default async function ComparePage({
  params,
}: {
  params: Promise<{ username: string; repo: string; range: string }>;
}) {
  const { username, repo, range } = await params;
  const [from, to] = decodeURIComponent(range).split("..");
  const short = (s: string) => (s && s.length > 7 ? s.slice(0, 7) : s);

  let diff: Diff | null = null;
  let failed = false;

  if (!from || !to) {
    failed = true;
  } else {
    try {
      diff = await getCompareDiff(username, repo, from, to);
    } catch {
      failed = true;
    }
  }

  if (failed || !diff) {
    return (
      <div className="p-6 text-base border border-base rounded-md bg-panel">
        Unable to load compare diff.
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

  return (
    <div className="border border-base rounded-md overflow-hidden bg-panel">
      <div className="px-4 py-3 border-b border-base flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm">
          <span className="font-mono bg-base px-2 py-1 rounded text-muted">
            {short(from)}..{short(to)}
          </span>
          <span className="text-muted">/</span>
          <span className="font-medium text-base">Compare</span>
        </div>
        <div className="flex items-center gap-4 text-sm">
          <span className="text-green-500">+{diff.additions}</span>
          <span className="text-red-500">-{diff.deletions}</span>
          <span className="text-muted">{diff.files_changed} files</span>
        </div>
      </div>
      <div className="p-4 space-y-4">
        {Array.isArray(diff.files) && diff.files.length > 0 ? (
          diff.files.map((f, i) => (
            <div key={i} className="border border-base rounded-md overflow-hidden">
              <div className="px-4 py-2 border-b border-base flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <span className="font-mono">{f.old_path}</span>
                  <span className="text-muted">→</span>
                  <span className="font-mono">{f.new_path}</span>
                  <span className="ml-2 text-xs px-2 py-0.5 rounded-full border border-base text-muted uppercase font-medium">
                    {f.status}
                  </span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-green-500">+{f.additions}</span>
                  <span className="text-red-500">-{f.deletions}</span>
                </div>
              </div>
              <DiffViewer patch={f.patch || ""} />
            </div>
          ))
        ) : (
          <DiffViewer patch={diff.content} />
        )}
      </div>
    </div>
  );
}
