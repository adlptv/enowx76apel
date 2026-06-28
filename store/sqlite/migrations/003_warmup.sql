CREATE TABLE IF NOT EXISTS warmup_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id  INTEGER NOT NULL,
    provider    TEXT NOT NULL,
    label       TEXT NOT NULL DEFAULT '',
    ok          INTEGER NOT NULL DEFAULT 0,
    outcome     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT '',
    request     TEXT NOT NULL DEFAULT '',
    response    TEXT NOT NULL DEFAULT '',
    usage       TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_warmup_created ON warmup_logs(created_at DESC);
