package db

const schemaSQL = `
CREATE TABLE IF NOT EXISTS entries (
    id           INTEGER PRIMARY KEY,
    content      TEXT NOT NULL,
    source_type  TEXT NOT NULL,
    timestamp    TEXT,
    project_path TEXT,
    project_hash TEXT,
    session_id   TEXT,
    message_id   TEXT,
    model        TEXT,
    source_file  TEXT NOT NULL,
    is_sidechain INTEGER DEFAULT 0,
    byte_offset  INTEGER,
    created_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_entries_timestamp ON entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_entries_project ON entries(project_hash);
CREATE INDEX IF NOT EXISTS idx_entries_source_type ON entries(source_type);
CREATE INDEX IF NOT EXISTS idx_entries_source_file ON entries(source_file);

CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    content,
    content=entries,
    content_rowid=id,
    tokenize='porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
    INSERT INTO entries_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, content) VALUES ('delete', old.id, old.content);
END;

CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, content) VALUES ('delete', old.id, old.content);
    INSERT INTO entries_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TABLE IF NOT EXISTS index_state (
    source_file  TEXT PRIMARY KEY,
    byte_offset  INTEGER NOT NULL,
    file_size    INTEGER NOT NULL,
    file_mtime   TEXT NOT NULL,
    entry_count  INTEGER DEFAULT 0,
    last_indexed TEXT,
    file_hash    TEXT
);

CREATE TABLE IF NOT EXISTS chunks (
    id           INTEGER PRIMARY KEY,
    source_file  TEXT NOT NULL,
    session_id   TEXT,
    project_hash TEXT,
    chunk_date   TEXT NOT NULL,
    first_ts     TEXT,
    last_ts      TEXT,
    entry_count  INTEGER,
    raw_gz       BLOB NOT NULL,
    UNIQUE(source_file, chunk_date)
);
`
