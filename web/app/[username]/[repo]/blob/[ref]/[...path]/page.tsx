import { getBlob, getBlame } from '@/lib/api';
import Link from 'next/link';
import { BlameLine } from '@/lib/types';

export default async function BlobPage({
  params,
  searchParams,
}: {
  params: Promise<{ username: string; repo: string; ref: string; path: string[] }>;
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const { username, repo, ref: refParam, path: pathSegments } = await params;
  const fullPath = [decodeURIComponent(refParam), ...(pathSegments || []).map(p => decodeURIComponent(p))].join('/');
  
  const { blame } = await searchParams;
  const isBlame = blame === 'true';

  let content = '';
  let blameData: BlameLine[] = [];
  let ref = '';
  let path = '';
  let failed = false;

  try {
    if (isBlame) {
      const data = await getBlame(username, repo, fullPath);
      blameData = data.blame;
      ref = data.ref;
      path = data.path;
    } else {
      const data = await getBlob(username, repo, fullPath);
      content = data.content;
      ref = data.ref;
      path = data.path;
    }
  } catch {
    failed = true;
  }

  if (failed) {
    return (
      <div className="p-6 text-sm text-base border border-base rounded-md bg-panel">
        Unable to load file.
        <div className="mt-2">
          <Link href={`/${username}/${repo}`} className="text-accent hover:underline">
            Back to repository
          </Link>
        </div>
      </div>
    );
  }

  const parentPath = path.split('/').slice(0, -1).join('/');
  const encodeSegments = (p: string) =>
    p
      .split('/')
      .filter(Boolean)
      .map((seg) => encodeURIComponent(seg))
      .join('/');
  const encodedParent = encodeSegments(parentPath);
  const lines = isBlame ? [] : content.split('\n');

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
            href={isBlame ? `/${username}/${repo}/blob/${ref}/${path}` : `/${username}/${repo}/blame/${ref}/${path}`}
            className="text-xs px-3 py-1 rounded transition-colors btn"
          >
            {isBlame ? 'Normal View' : 'Blame'}
          </Link>
          <Link 
            href={`/${username}/${repo}/commits/${ref}/${path}`}
            className="text-xs px-2 py-1 rounded transition-colors btn"
          >
            History
          </Link>
          <Link 
            href={encodedParent ? `/${username}/${repo}/tree/${encodeURIComponent(ref)}/${encodedParent}` : `/${username}/${repo}/tree/${encodeURIComponent(ref)}`}
            className="text-xs px-2 py-1 rounded transition-colors btn"
          >
            View Parent
          </Link>
        </div>
      </div>
      <div className="overflow-x-auto text-sm font-mono leading-6 bg-panel">
        <table className="w-full border-collapse">
          <tbody>
            {isBlame ? (
              blameData.map((line, i) => (
                <tr key={i}>
                  <td className="w-48 px-2 text-xs text-muted border-r border-base truncate" title={line.commit}>
                    {line.commit.substring(0, 7)} <span className="text-zinc-400">|</span> {line.author}
                  </td>
                  <td className="w-12 text-right select-none text-muted bg-panel pr-4 border-r border-base py-0.5">
                    {line.line_no}
                  </td>
                  <td className="pl-4 whitespace-pre text-base py-0.5">
                    {line.content}
                  </td>
                </tr>
              ))
            ) : (
              lines.map((line, i) => (
                <tr key={i}>
                  <td className="w-12 text-right select-none text-muted bg-panel pr-4 border-r border-base py-0.5">
                    {i + 1}
                  </td>
                  <td className="pl-4 whitespace-pre text-base py-0.5">
                    {line}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
