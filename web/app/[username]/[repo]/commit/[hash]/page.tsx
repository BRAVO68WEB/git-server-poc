import { getDiff } from '@/lib/api';
import Link from 'next/link';

export default async function CommitPage({ params }: { params: Promise<{ username: string; repo: string; hash: string }> }) {
  const { username, repo, hash } = await params;
  const diff = await getDiff(username, repo, hash);

  const lines = diff.content.split('\n');

  const getLineClass = (line: string) => {
    if (line.startsWith('+++') || line.startsWith('---')) return 'text-sky-400';
    if (line.startsWith('diff --git')) return 'text-sky-300';
    if (line.startsWith('@@')) return 'text-amber-300';
    if (line.startsWith('+')) return 'text-green-400 bg-green-950/40';
    if (line.startsWith('-')) return 'text-red-400 bg-red-950/40';
    return 'text-base';
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-6">
        <Link href={`/${username}/${repo}/commits`} className="text-accent hover:underline mb-2 inline-block">
          &larr; Back to Commits
        </Link>
        <h1 className="text-2xl font-bold mb-2 text-base">Commit {hash.substring(0, 7)}</h1>
      </div>

      <div className="bg-panel rounded-lg border border-base overflow-hidden">
        <div className="px-4 py-2 border-b border-base font-mono text-sm">
          Diff
        </div>
        <pre className="p-4 overflow-x-auto text-sm font-mono whitespace-pre bg-base text-base leading-relaxed">
          {lines.map((line, idx) => (
            <span key={idx} className={getLineClass(line)}>
              {line || '\u00a0'}
              {'\n'}
            </span>
          ))}
        </pre>
      </div>
    </div>
  );
}
