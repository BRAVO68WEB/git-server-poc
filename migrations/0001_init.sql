CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  role TEXT NOT NULL,
  disabled BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ssh_keys (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id),
  name TEXT NOT NULL,
  pubkey TEXT NOT NULL,
  fingerprint TEXT UNIQUE,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS repos (
  id BIGSERIAL PRIMARY KEY,
  owner_id BIGINT REFERENCES users(id),
  name TEXT NOT NULL,
  visibility TEXT NOT NULL,
  default_branch TEXT,
  archived BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE(owner_id, name)
);

CREATE TABLE IF NOT EXISTS repo_members (
  repo_id BIGINT REFERENCES repos(id),
  user_id BIGINT REFERENCES users(id),
  role TEXT NOT NULL,
  PRIMARY KEY (repo_id, user_id)
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  actor_id BIGINT REFERENCES users(id),
  action TEXT NOT NULL,
  repo_id BIGINT REFERENCES repos(id),
  ip INET,
  ts TIMESTAMPTZ DEFAULT now(),
  meta JSONB
);
