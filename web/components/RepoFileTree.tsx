import Link from 'next/link';
import { FileEntry } from '@/lib/types';

export function RepoFileTree({ 
  owner, 
  name, 
  currentRef, 
  path, 
  entries 
}: { 
  owner: string; 
  name: string; 
  currentRef: string; 
  path: string; 
  entries: FileEntry[] 
}) {
  const list = Array.isArray(entries) ? entries : [];
  const sorted = [...list].sort((a, b) => {
    if (a.type === 'tree' && b.type === 'blob') return -1;
    if (a.type === 'blob' && b.type === 'tree') return 1;
    return a.name.localeCompare(b.name);
  });

  const parentPath = path.split('/').slice(0, -1).join('/');
  const isRoot = path === '';

  // Helper to encode path segments properly if needed, but usually basic URL encoding is handled by Next.js Link.
  // We construct the URL manually.
  const encodeSegments = (p: string) =>
    p
      .split('/')
      .filter(Boolean)
      .map((seg) => encodeURIComponent(seg))
      .join('/');
  const encodedParent = encodeSegments(parentPath);
  const baseTree = `/${owner}/${name}/tree/${encodeURIComponent(currentRef)}`;

  return (
    <div className="border border-base rounded-md overflow-hidden bg-panel">
      <div className="px-4 py-3 border-b border-base flex items-center justify-between">
         <div className="flex items-center gap-2 text-sm">
            <span className="font-mono bg-base px-2 py-1 rounded text-muted">{currentRef}</span>
            <span className="text-muted">/</span>
            <span className="font-medium text-base">{path || ''}</span>
         </div>
      </div>
      <table className="w-full text-sm text-left">
        <tbody>
          {!isRoot && (
             <tr className="border-b border-base hover:bg-base transition-colors">
               <td className="px-4 py-2" colSpan={3}>
                 <Link href={encodedParent ? `${baseTree}/${encodedParent}` : `${baseTree}`} className="text-accent font-bold block w-full">
                   ..
                 </Link>
               </td>
             </tr>
          )}
          {sorted.map((entry) => {
            const relativeName = path && entry.name.startsWith(path + '/')
              ? entry.name.slice(path.length + 1)
              : entry.name;
            const entryPath = path ? `${path}/${relativeName}` : relativeName;
            const encodedEntryPath = encodeSegments(entryPath);
            const displayName = relativeName.split('/').filter(Boolean).pop() || relativeName;
            const href =
              entry.type === 'tree'
                ? `${baseTree}/${encodedEntryPath}`
                : `/${owner}/${name}/blob/${encodeURIComponent(currentRef)}/${encodedEntryPath}`;
            
            return (
              <tr key={entry.name} className="border-b last:border-0 border-base hover:bg-base transition-colors">
                <td className="px-4 py-2 w-8 text-muted text-center">
                   {entry.type === 'tree' ? '📁' : '📄'}
                </td>
                <td className="px-4 py-2">
                  <Link href={href} className="text-base hover:text-accent hover:underline block">
                    {displayName}
                  </Link>
                </td>
                <td className="px-4 py-2 text-right text-muted text-xs font-mono">
                  {entry.hash.substring(0, 7)}
                </td>
              </tr>
            );
          })}
          {sorted.length === 0 && (
            <tr>
                <td colSpan={3} className="px-4 py-8 text-center text-muted">
                    Empty directory
                </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
