-- Recreate ocr_draft_entries without the broken foreign key to rates(category).
-- rates.category lacks a UNIQUE constraint, so the FK causes "foreign key mismatch"
-- errors on any DML that touches ocr_draft_entries.

CREATE TABLE IF NOT EXISTS ocr_draft_entries_new (
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
    FOREIGN KEY (session_id) REFERENCES ocr_import_sessions(id) ON DELETE CASCADE
);

INSERT INTO ocr_draft_entries_new SELECT * FROM ocr_draft_entries;

DROP TABLE IF EXISTS ocr_draft_entries;

ALTER TABLE ocr_draft_entries_new RENAME TO ocr_draft_entries;

CREATE INDEX IF NOT EXISTS idx_ocr_drafts_session ON ocr_draft_entries(session_id);
