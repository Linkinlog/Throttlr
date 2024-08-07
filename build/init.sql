CrEaTe TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CrEaTe TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    key TEXT NOT NULL,
    valid BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CrEaTe TABLE IF NOT EXISTS endpoints (
    id SERIAL PRIMARY KEY,
    api_key_id integer NOT NULL,
    original_url TEXT NOT NULL,
    throttlr_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX throttlr_url_idx ON endpoints(throttlr_url);
CrEaTe TABLE IF NOT EXISTS buckets (
    id SERIAL PRIMARY KEY,
    endpoint_id integer NOT NULL,
    max integer NOT NULL,
    interval integer NOT NULL,
    window_opened_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
