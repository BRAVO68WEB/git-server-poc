import { getBlame } from '@/lib/api';
import Link from 'next/link';

export default async function BlamePage({
  params,
}: {
  params: Promise<{ username: string; repo: string; ref: string; path: string[] }>;
}) {
  const { username, repo, ref: refParam, path: pathSegments } = await params;
  const fullPath = [decodeURIComponent(refParam), ...(pathSegments || []).map(p => decodeURIComponent(p))].join('/');

  let ref = '';
  let path = '';
  let blameData: Array<{ line_no: number; commit: string; author: string; date: string; content: string }> = [];
  let failed = false;

  try {
    const data = await getBlame(username, repo, fullPath);
    ref = data.ref;
    path = data.path;
    blameData = data.blame || [];
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
        Unable to load blame.
        <div className="mt-2">
          <Link href={`/${username}/${repo}`} className="text-accent hover:underline">
            Back to repository
          </Link>
        </div>
      </div>
    );
  }

  const parentPath = path.split('/').slice(0, -1).join('/');

  return (
    <div className="border border-base rounded-md overflow-hidden bg-panel">
      <div className="px-4 py-3 border-b border-base flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm">
          <span className="font-mono bg-base px-2 py-1 rounded text-muted">{ref}</span>
          <span className="text-muted">/</span>
          <span className="font-medium text-base">{path}</span>
        </div>
        <div className="flex items-center gap-2">
          <Link 
            href={`/${username}/${repo}/blob/${ref}/${path}`}
            className="text-xs px-2 py-1 rounded btn"
          >
            Normal View
          </Link>
          <Link 
            href={`/${username}/${repo}/tree/${ref}/${parentPath}`}
            className="text-xs px-2 py-1 rounded btn"
          >
            View Parent
          </Link>
        </div>
      </div>
      <div className="overflow-x-auto text-sm font-mono leading-6 bg-panel">
        <table className="w-full border-collapse">
          <tbody>
            {blameData.map((line, i) => (
              <tr key={i}>
                <td className="w-48 px-2 text-xs text-muted border-r border-base truncate" title={line.commit}>
                  {line.commit.substring(0, 7)} <span className="text-muted">|</span> {line.author}
                </td>
                <td className="w-12 text-right select-none text-muted bg-panel pr-4 border-r border-base py-0.5">
                  {line.line_no}
                </td>
                <td className="pl-4 whitespace-pre text-base py-0.5">
                  {line.content}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
