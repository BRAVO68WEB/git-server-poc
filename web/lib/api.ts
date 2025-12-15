import { Repo, FileEntry, Commit, Branch, Diff, BlameLine } from './types';

const API_URL = process.env.API_URL || 'http://localhost:8094';

export async function getRepos(): Promise<Repo[]> {
  const res = await fetch(`${API_URL}/api/repos`, { cache: 'no-store' });
  if (!res.ok) {
    console.error('Failed to fetch repos', await res.text());
    return [];
  }
  return res.json();
}

export async function getRepo(owner: string, name: string): Promise<Repo> {
  const res = await fetch(`${API_URL}/api/repos/${owner}/${name}`, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch repo');
  return res.json();
}

export async function getTree(owner: string, name: string, urlPath: string): Promise<{ ref: string; path: string; entries: FileEntry[] }> {
  const url = `${API_URL}/api/repos/${owner}/${name}/tree/${urlPath}`;
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch tree');
  return res.json();
}

export async function getBlob(owner: string, name: string, urlPath: string): Promise<{ ref: string; path: string; content: string }> {
  const url = `${API_URL}/api/repos/${owner}/${name}/blob/${urlPath}`;
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch blob');
  return res.json();
}

export async function getCommits(owner: string, name: string, urlPath: string): Promise<{ ref: string; path: string; commits: Commit[] }> {
  const url = `${API_URL}/api/repos/${owner}/${name}/commits/${urlPath}`;
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch commits');
  return res.json();
}

export async function getBranches(owner: string, name: string): Promise<Branch[]> {
  const url = `${API_URL}/api/repos/${owner}/${name}/branches`;
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch branches');
  return res.json();
}

export async function getDiff(owner: string, name: string, hash: string): Promise<Diff> {
  const url = `${API_URL}/api/repos/${owner}/${name}/diff/${hash}`;
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch diff');
  return res.json();
}

export async function getBlame(owner: string, name: string, urlPath: string): Promise<{ ref: string; path: string; blame: BlameLine[] }> {
  const url = `${API_URL}/api/repos/${owner}/${name}/blame/${urlPath}`;
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch blame');
  return res.json();
}
