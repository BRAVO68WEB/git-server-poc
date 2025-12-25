import { listTags } from "@/lib/api";
import Link from "next/link";

export default async function TagsPage({
  params,
}: {
  params: Promise<{ username: string; repo: string }>;
}) {
  const { username, repo } = await params;
  let tags: Array<{
    name: string;
    hash: string;
    message?: string;
    tagger?: string;
    is_annotated: boolean;
  }> = [];
  let failed = false;

  try {
    const response = await listTags(username, repo);
    tags = response.tags;
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
        Unable to load tags.
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
      <div className="px-4 py-3 border-b border-base">
        <span className="font-semibold text-base">Tags ({tags.length})</span>
      </div>
      <div className="divide-y divide-[var(--border-base)]">
        {tags.map((tag) => (
          <div
            key={tag.name}
            className="p-4 flex items-center justify-between hover:bg-base"
          >
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <span className="font-mono text-sm text-base font-semibold">
                  {tag.name}
                </span>
                {tag.is_annotated && (
                  <span className="text-xs px-2 py-0.5 rounded border border-base text-muted">
                    annotated
                  </span>
                )}
              </div>
              {tag.message && (
                <p className="text-sm text-muted line-clamp-1">{tag.message}</p>
              )}
              {tag.tagger && (
                <p className="text-xs text-muted">{tag.tagger}</p>
              )}
            </div>
            <div className="flex items-center gap-4">
              <span className="font-mono text-xs text-muted">
                {tag.hash.substring(0, 7)}
              </span>
              <div className="flex items-center gap-2">
                <Link
                  href={`/${username}/${repo}/tree/${tag.name}`}
                  className="text-xs px-2 py-1 rounded btn"
                >
                  Browse
                </Link>
                <Link
                  href={`/${username}/${repo}/commit/${tag.hash}`}
                  className="text-xs px-2 py-1 rounded btn"
                >
                  View Commit
                </Link>
              </div>
            </div>
          </div>
        ))}
        {tags.length === 0 && (
          <div className="p-6 text-center text-muted">
            No tags found in this repository.
          </div>
        )}
      </div>
    </div>
  );
}
