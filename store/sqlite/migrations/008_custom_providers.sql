-- User-added OpenAI/Anthropic-compatible providers. Each becomes a live provider
-- with its own prefix. Registered into the registry at boot + on change.
CREATE TABLE IF NOT EXISTS custom_providers (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL,
    prefix        TEXT NOT NULL UNIQUE,
    format        TEXT NOT NULL DEFAULT 'openai',   -- openai | anthropic
    base_url      TEXT NOT NULL,
    default_model TEXT NOT NULL DEFAULT '',
    models        TEXT NOT NULL DEFAULT '[]',       -- JSON array of {id,name}
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
