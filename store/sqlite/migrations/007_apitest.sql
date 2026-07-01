-- API Test (Postman-style dev tool) persistence: collections of saved requests,
-- environments (variable sets), and a run history. All local to this instance.
CREATE TABLE IF NOT EXISTS apitest_collections (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    sort       INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS apitest_requests (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    collection_id INTEGER NOT NULL REFERENCES apitest_collections(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    method        TEXT NOT NULL DEFAULT 'GET',
    url           TEXT NOT NULL DEFAULT '',
    headers       TEXT NOT NULL DEFAULT '[]',  -- [{key,value,on}]
    query         TEXT NOT NULL DEFAULT '[]',  -- [{key,value,on}]
    body          TEXT NOT NULL DEFAULT '',
    body_type     TEXT NOT NULL DEFAULT 'none', -- none|json|form|multipart|raw|graphql
    auth          TEXT NOT NULL DEFAULT '{}',   -- {type,token,username,password,key,value,in}
    sort          INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS apitest_environments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    vars       TEXT NOT NULL DEFAULT '[]',  -- [{key,value}]
    active     INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS apitest_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    method      TEXT NOT NULL,
    url         TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    at          TEXT NOT NULL DEFAULT (datetime('now'))
);
