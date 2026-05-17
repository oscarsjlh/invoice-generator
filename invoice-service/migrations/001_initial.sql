CREATE TABLE IF NOT EXISTS entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    date       TEXT NOT NULL,
    category   TEXT NOT NULL,
    hours      REAL NOT NULL CHECK (hours > 0),
    notes      TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rates (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    category   TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date   TEXT NOT NULL,
    rate       REAL NOT NULL CHECK (rate > 0),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invoices (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_number   TEXT NOT NULL UNIQUE,
    month            TEXT NOT NULL,
    category         TEXT NOT NULL,
    invoice_date     TEXT NOT NULL,
    due_date         TEXT NOT NULL,
    subtotal         REAL NOT NULL,
    total            REAL NOT NULL,
    business_name    TEXT NOT NULL DEFAULT '',
    business_address TEXT NOT NULL DEFAULT '',
    bank_name        TEXT NOT NULL DEFAULT '',
    account_name     TEXT NOT NULL DEFAULT '',
    account_number   TEXT NOT NULL DEFAULT '',
    sort_code        TEXT NOT NULL DEFAULT '',
    payment_terms    TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invoice_lines (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL,
    category   TEXT NOT NULL,
    hours      REAL NOT NULL,
    rate       REAL NOT NULL,
    amount     REAL NOT NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_entries_date ON entries(date);
CREATE INDEX IF NOT EXISTS idx_entries_category ON entries(category);
CREATE INDEX IF NOT EXISTS idx_rates_lookup ON rates(category, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_invoices_created_at ON invoices(created_at DESC);

INSERT INTO settings (key, value) VALUES
    ('business_name', ''),
    ('business_address', ''),
    ('bank_name', ''),
    ('account_name', ''),
    ('account_number', ''),
    ('sort_code', ''),
    ('payment_terms', 'Payment due within 30 days.'),
    ('default_due_days', '30')
ON CONFLICT(key) DO NOTHING;
