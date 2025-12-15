export interface Repo {
  owner: string;
  name: string;
  description: string;
  visibility: string;
}

export interface FileEntry {
  mode: string;
  type: string;
  hash: string;
  name: string;
  size?: number;
}

export interface Commit {
  hash: string;
  author: string;
  date: string;
  message: string;
}

export interface Branch {
  name: string;
  is_head: boolean;
}

export interface Diff {
  content: string;
}

export interface BlameLine {
  line_no: number;
  commit: string;
  author: string;
  date: string;
  content: string;
}
