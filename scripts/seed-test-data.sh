#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_PATH="${1:-${DATABASE_PATH:-$ROOT_DIR/invoice-service/data/invoices.db}}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$ROOT_DIR/invoice-service/migrations}"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 is required to seed test data" >&2
  exit 1
fi

mkdir -p "$(dirname "$DB_PATH")"

for migration in "$MIGRATIONS_DIR"/*.sql; do
  sqlite3 "$DB_PATH" < "$migration"
done

sqlite3 "$DB_PATH" <<'SQL'
PRAGMA foreign_keys = ON;

BEGIN;

INSERT INTO settings (key, value) VALUES
  ('business_name', 'Northstar Studio Ltd'),
  ('business_address', '18 Market Street
Bristol
BS1 1AA'),
  ('bank_name', 'Example Bank'),
  ('account_name', 'Northstar Studio Ltd'),
  ('account_number', '12345678'),
  ('sort_code', '12-34-56'),
  ('payment_terms', 'Payment due within 14 days.'),
  ('default_due_days', '14'),
  ('customer_name', 'Acme Operations'),
  ('customer_title', 'Finance Team'),
  ('customer_email', 'finance@example.test'),
  ('customer_address', '4 Client Road
Manchester
M1 2AB'),
  ('customer_postal_code', 'M1 2AB'),
  ('customer_city', 'Manchester')
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

DELETE FROM entries WHERE notes LIKE '[seed] %';
DELETE FROM rates WHERE category IN ('Consulting', 'Support', 'Training');

INSERT INTO rates (category, start_date, end_date, rate) VALUES
  ('Consulting', '2026-01-01', '', 95.00),
  ('Support', '2026-01-01', '', 70.00),
  ('Training', '2026-01-01', '', 85.00);

INSERT INTO entries (date, category, hours, notes) VALUES
  ('2026-05-04', 'Consulting', 6.5, '[seed] dashboard workflow review'),
  ('2026-05-05', 'Consulting', 7.0, '[seed] invoice generation tracing'),
  ('2026-05-06', 'Support', 3.0, '[seed] customer support follow-up'),
  ('2026-05-07', 'Training', 4.5, '[seed] onboarding session'),
  ('2026-05-11', 'Consulting', 6.0, '[seed] OCR import testing'),
  ('2026-05-12', 'Support', 2.5, '[seed] bug triage'),
  ('2026-05-13', 'Consulting', 7.5, '[seed] PDF rendering checks'),
  ('2026-04-08', 'Consulting', 5.0, '[seed] previous month planning'),
  ('2026-04-09', 'Support', 2.0, '[seed] previous month support'),
  ('2026-04-10', 'Training', 3.5, '[seed] previous month workshop');

COMMIT;
SQL

echo "Seeded test data into $DB_PATH"
echo "Run the app with DATABASE_PATH=$DB_PATH when AUTH_ENABLED=false."
echo "For auth-enabled local users, pass that user's SQLite DB path, for example:"
echo "  $0 invoice-service/data/users/1.db"
