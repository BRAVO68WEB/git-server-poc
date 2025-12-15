'use client';

import { useRouter, useParams, usePathname } from 'next/navigation';
import { Branch } from '../lib/types';

export default function BranchSelector({ branches }: { branches: Branch[] }) {
  const router = useRouter();
  const params = useParams();
  const pathname = usePathname();
  const currentRef = (params.ref as string) || 'HEAD';

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newRef = e.target.value;
    // Replace the current ref in the path with the new ref
    // Path structure: /[username]/[repo]/tree/[ref]/...
    // Or /[username]/[repo]/blob/[ref]/...
    // Or /[username]/[repo]/commits/[ref]
    
    // Simple heuristic: split by ref and join? 
    // This is tricky because ref might be part of path? No, ref is a segment.
    // Let's assume standard routes.
    
    const parts = pathname.split('/');
    // ['', username, repo, action, ref, ...path]
    if (parts.length >= 5) {
      parts[4] = newRef;
      router.push(parts.join('/'));
    } else {
      // Maybe root repo page which defaults to default branch?
      // Redirect to tree view with new branch
      router.push(`/${params.username}/${params.repo}/tree/${newRef}`);
    }
  };

  return (
    <div className="relative inline-block text-left mr-4">
      <select 
        value={decodeURIComponent(currentRef)}
        onChange={handleChange}
        className="block appearance-none w-full bg-panel border border-base px-4 py-2 pr-8 rounded leading-tight text-base"
      >
        {branches.map((b) => (
          <option key={b.name} value={b.name} className="bg-panel text-base">
            {b.name}
          </option>
        ))}
      </select>
    </div>
  );
}
