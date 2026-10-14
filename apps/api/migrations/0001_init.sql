CREATE TABLE users (
  id TEXT PRIMARY KEY,
  google_sub TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  name TEXT,
  avatar_url TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX sessions_user_id ON sessions(user_id);

-- id es el SHA-256 del .ogg, igual que el assetId de la v1.
CREATE TABLE sounds (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  duration_ms INTEGER NOT NULL,
  audio_key TEXT NOT NULL,
  image_key TEXT,
  image_type TEXT,
  status TEXT NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'hidden')),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX sounds_status_created ON sounds(status, created_at DESC);

CREATE TABLE user_sounds (
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  sound_id TEXT NOT NULL REFERENCES sounds(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  PRIMARY KEY (user_id, sound_id)
);

CREATE TABLE jobs (
  id TEXT PRIMARY KEY,
  source_url TEXT NOT NULL,
  image_url TEXT,
  title TEXT NOT NULL,
  trim_from TEXT,
  trim_to TEXT,
  stereo INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
  error TEXT,
  sound_id TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX jobs_created ON jobs(created_at DESC);
