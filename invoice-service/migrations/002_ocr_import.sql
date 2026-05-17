CREATE TABLE IF NOT EXISTS ocr_import_sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    state      TEXT NOT NULL DEFAULT 'uploaded',
    error_msg  TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ocr_session_images (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    file_name  TEXT NOT NULL,
    file_path  TEXT NOT NULL,
    file_size  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES ocr_import_sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ocr_draft_entries (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id           INTEGER NOT NULL,
    date_raw             TEXT NOT NULL DEFAULT '',
    date_normalized      TEXT NOT NULL DEFAULT '',
    category_raw         TEXT NOT NULL DEFAULT '',
    category_normalized  TEXT NOT NULL DEFAULT '',
    hours_raw            TEXT NOT NULL DEFAULT '',
    hours_normalized     REAL NOT NULL DEFAULT 0.0,
    notes_raw            TEXT NOT NULL DEFAULT '',
    notes_normalized     TEXT NOT NULL DEFAULT '',
    confidence           REAL NOT NULL DEFAULT 0.0,
    needs_review         INTEGER NOT NULL DEFAULT 0,
    confirmed            INTEGER NOT NULL DEFAULT 0,
    created_at           TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES ocr_import_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (category_normalized) REFERENCES rates(category)
);

CREATE INDEX IF NOT EXISTS idx_ocr_sessions_state ON ocr_import_sessions(state);
CREATE INDEX IF NOT EXISTS idx_ocr_drafts_session ON ocr_draft_entries(session_id);
CREATE INDEX IF NOT EXISTS idx_ocr_images_session ON ocr_session_images(session_id);
