# Missing `show_payment_due` Column — Root Cause Analysis

## Symptom

Generating an invoice in the web UI fails with:

> Could not generate invoice: insert invoice: SQL logic error: table invoices has no column named show_payment_due (1)

## The Three Bugs

### Bug 1 — Untracked migration file edited after application

The file `invoice-service/migrations/008_invoice_utr_and_service_dates.sql` was **never committed to git** (untracked). It was originally created with only two ALTER TABLE statements:

```sql
-- Original content (applied to the database)
ALTER TABLE invoices ADD COLUMN business_utr TEXT NOT NULL DEFAULT '';
ALTER TABLE invoice_lines ADD COLUMN service_dates TEXT NOT NULL DEFAULT '';
```

The Docker container was rebuilt, picked up this file, and ran it. Both statements succeeded. The migration was recorded in `schema_migrations`.

Later, the file was edited on disk to add the third line:

```sql
-- Line added AFTER migration was already recorded
ALTER TABLE invoices ADD COLUMN show_payment_due INTEGER NOT NULL DEFAULT 1;
```

But because the filename was already in `schema_migrations`, the runner **skipped it permanently**. The column was never added.

This is a classic migration smell: **editing an already-applied migration file instead of creating a new one.** The proper fix would be `009_add_show_payment_due.sql`, but since this is development and the database can be repaired, we removed the stale `schema_migrations` record and let it re-run.

---

### Bug 2 — No transaction wrapping on multi-statement migrations

The `Migrate()` function in `internal/db/db.go` executed each statement from a migration file **individually without a transaction**. SQLite auto-commits DDL, so if a migration had 3 ALTER TABLE statements and statement 2 failed:

1. Statement 1 succeeds → committed
2. Statement 2 fails → error returned
3. Statement 3 never runs
4. `recordMigration()` is never called (migration not recorded)
5. On next startup: migration runs again, statement 1 fails with "duplicate column name", runner bails out, `os.Exit(1)` kills the process

**The database is left in a permanently broken state** — half-migrated, can't retry, can't proceed past the broken file.

**Fix applied:** All statements from a single migration file are now wrapped in `DB.Begin()` / `Tx.Commit()`. If any statement fails, the transaction rolls back and the database is untouched.

---

### Bug 3 — No idempotency for ALTER TABLE ADD COLUMN

Even with transactions, if a migration partially succeeded *before* the transaction fix was deployed, the database is already dirty. The runner would hit "duplicate column name" on retry and fail.

**Fix applied:** Added `isDuplicateColumnError()` — when an `ALTER TABLE` statement fails with "duplicate column name", it is silently skipped. This makes migrations safe to re-run against partially-migrated databases.

## Full Timeline

| Time (UTC) | Event |
|---|---|
| 14:34 | `008_invoice_utr_and_service_dates.sql` created on disk (2 statements, no `show_payment_due`) |
| 14:40 | Docker container starts, migration 008 runs (adds `business_utr` + `service_dates`), recorded in `schema_migrations` |
| 15:06 | File edited: third `ALTER TABLE` for `show_payment_due` added to disk |
| 15:19 | Container restarted — migration 008 already recorded → **skipped**. `show_payment_due` never added |
| 15:20 | User hits "generate invoice" → `SQL logic error: table invoices has no column named show_payment_due` |
| 15:27 | Manual fix: `sqlite3 data/invoices.db "ALTER TABLE invoices ADD COLUMN show_payment_due ..."`, migration record deleted, container recreated → migration 008 re-runs successfully |

## Files Changed

| File | Change |
|---|---|
| `internal/db/db.go` | Added `isDuplicateColumnError()`, wrapped `Migrate()` statements in a transaction, ALTER TABLE failures are skipped if column already exists |
| `internal/db/authdb.go` | Same transaction + idempotent ALTER TABLE fix for the auth database migrator |
| `migrations/008_invoice_utr_and_service_dates.sql` | Staged for git tracking (was untracked) |

## Remote Fix Command

If the same error appears on the remote deployment, run on the host (volume is mounted):

```bash
sqlite3 ./data/invoices.db "ALTER TABLE invoices ADD COLUMN show_payment_due INTEGER NOT NULL DEFAULT 1;"
```

## Prevention Checklist

- [ ] All migration files are tracked in git
- [ ] Migrations are wrapped in transactions (done in `db.go` / `authdb.go`)
- [ ] ALTER TABLE ADD COLUMN is idempotent (done — duplicate columns are silently skipped)
- [ ] **Never edit an already-applied migration** — always create a new one
- [ ] Consider adding migration checksum verification to detect tampering (goose does this automatically)
