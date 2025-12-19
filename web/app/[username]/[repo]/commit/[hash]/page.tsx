import { getDiff } from "@/lib/api";
import Link from "next/link";

export default async function CommitPage({
  params,
}: {
  params: Promise<{ username: string; repo: string; hash: string }>;
}) {
  const { username, repo, hash } = await params;
  const diff = await getDiff(username, repo, hash);

  const getLineClass = (line: string) => {
    if (line.startsWith("+++") || line.startsWith("---")) return "text-sky-400";
    if (line.startsWith("diff --git")) return "text-sky-300 font-semibold";
    if (line.startsWith("@@")) return "text-amber-300 bg-amber-950/20";
    if (line.startsWith("+")) return "text-green-400 bg-green-950/40";
    if (line.startsWith("-")) return "text-red-400 bg-red-950/40";
    return "text-base";
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case "added":
        return "bg-green-600/20 text-green-400 border-green-600/30";
      case "deleted":
        return "bg-red-600/20 text-red-400 border-red-600/30";
      case "renamed":
        return "bg-blue-600/20 text-blue-400 border-blue-600/30";
      default:
        return "bg-amber-600/20 text-amber-400 border-amber-600/30";
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case "added":
        return "Added";
      case "deleted":
        return "Deleted";
      case "renamed":
        return "Renamed";
      default:
        return "Modified";
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-6">
        <Link
          href={`/${username}/${repo}/commits`}
          className="text-accent hover:underline mb-2 inline-flex items-center gap-1"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="m15 18-6-6 6-6" />
          </svg>
          Back to Commits
        </Link>
        <h1 className="text-2xl font-bold mb-2 text-base">
          Commit{" "}
          <code className="text-accent bg-accent/10 px-2 py-1 rounded">
            {hash.substring(0, 7)}
          </code>
        </h1>
      </div>

      {/* Stats Summary */}
      <div className="bg-panel rounded-lg border border-base p-4 mb-6">
        <div className="flex flex-wrap items-center gap-4 text-sm">
          <div className="flex items-center gap-2">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="text-muted"
            >
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
              <polyline points="14 2 14 8 20 8" />
            </svg>
            <span className="text-muted">
              <strong className="text-base">{diff.files_changed}</strong>{" "}
              {diff.files_changed === 1 ? "file" : "files"} changed
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-green-400 font-mono">+{diff.additions}</span>
            <span className="text-muted">/</span>
            <span className="text-red-400 font-mono">-{diff.deletions}</span>
          </div>
        </div>
      </div>

      {/* File List */}
      {diff.files && diff.files.length > 0 && (
        <div className="bg-panel rounded-lg border border-base mb-6">
          <div className="px-4 py-3 border-b border-base">
            <h2 className="font-semibold text-base">Changed Files</h2>
          </div>
          <div className="divide-y divide-base">
            {diff.files.map((file, idx) => (
              <div
                key={idx}
                className="px-4 py-2 flex items-center justify-between hover:bg-base/50 transition-colors"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span
                    className={`text-xs px-2 py-0.5 rounded border ${getStatusBadgeClass(file.status)}`}
                  >
                    {getStatusLabel(file.status)}
                  </span>
                  <span className="font-mono text-sm text-base truncate">
                    {file.status === "renamed"
                      ? `${file.old_path} → ${file.new_path}`
                      : file.new_path}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs font-mono flex-shrink-0 ml-4">
                  {file.additions > 0 && (
                    <span className="text-green-400">+{file.additions}</span>
                  )}
                  {file.deletions > 0 && (
                    <span className="text-red-400">-{file.deletions}</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Full Diff */}
      <div className="bg-panel rounded-lg border border-base overflow-hidden">
        <div className="px-4 py-3 border-b border-base flex items-center justify-between">
          <h2 className="font-semibold text-base">Diff</h2>
          <span className="text-xs text-muted font-mono">{hash}</span>
        </div>
        <div className="overflow-x-auto">
          <pre className="p-4 text-sm font-mono whitespace-pre bg-base text-base leading-relaxed">
            {diff.content.split("\n").map((line, idx) => (
              <div key={idx} className={`${getLineClass(line)} px-1 -mx-1`}>
                {line || "\u00a0"}
              </div>
            ))}
          </pre>
        </div>
      </div>
    </div>
  );
}
