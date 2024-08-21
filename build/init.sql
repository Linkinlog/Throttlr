CrEaTe TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX users_email_idx ON users (email);

CREATE UNIQUE INDEX users_id_idx ON users (id);

CrEaTe TABLE IF NOT EXISTS api_keys (
  id SERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  key TEXT NOT NULL,
  valid BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX api_keys_user_id_idx ON api_keys (user_id);

CREATE UNIQUE INDEX api_keys_key_idx ON api_keys (key);

CrEaTe TABLE IF NOT EXISTS endpoints (
  id SERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  bucket_id integer NOT NULL,
  original_url TEXT NOT NULL,
  throttlr_url TEXT NOT NULL UNIQUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX endpoints_user_id_idx ON endpoints (user_id);

CREATE INDEX endpoints_throttlr_url_idx ON endpoints (throttlr_url);

CREATE UNIQUE INDEX endpoints_bucket_id_idx ON endpoints (bucket_id);

CrEaTe TABLE IF NOT EXISTS buckets (
  id SERIAL PRIMARY KEY,
  max integer NOT NULL,
  interval integer NOT NULL,
  window_opened_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX buckets_id_idx ON buckets (id);
