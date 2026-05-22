package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"invoice-app/internal/config"
	"invoice-app/internal/db"

	_ "modernc.org/sqlite"
)

func main() {
	fromDB := flag.String("from", "data/invoices.db", "path to the legacy SQLite database")
	userID := flag.Int64("user-id", 0, "user ID to migrate data to")
	username := flag.String("username", "", "lookup user by username instead of ID")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *userID == 0 && *username == "" {
		logger.Error("missing required flags", "usage", "go run ./cmd/migrate-to-user --user-id 1 --from data/invoices.db")
		os.Exit(1)
	}

	cfg := config.Load()

	authDB, err := db.OpenAuthDB(cfg.AuthDBPath)
	if err != nil {
		logger.Error("open auth database", "error", err)
		os.Exit(1)
	}
	defer authDB.Close()

	var uid int64
	if *userID != 0 {
		uid = *userID
		user, err := authDB.GetUserByID(uid)
		if err != nil {
			logger.Error("get user by ID", "user_id", uid, "error", err)
			os.Exit(1)
		}
		if user == nil {
			logger.Error("user not found", "user_id", uid)
			os.Exit(1)
		}
	} else {
		user, err := authDB.GetUserByUsernameForAuth(*username)
		if err != nil {
			logger.Error("get user by username", "username", *username, "error", err)
			os.Exit(1)
		}
		if user == nil {
			logger.Error("user not found", "username", *username)
			os.Exit(1)
		}
		uid = user.ID
	}

	logger.Info("opening source database", "path", *fromDB)

	srcDB, err := sql.Open("sqlite", *fromDB)
	if err != nil {
		logger.Error("open source database", "error", err)
		os.Exit(1)
	}
	defer srcDB.Close()

	multiStore := db.NewMultiStore(cfg.UserDBDir, cfg.MigrationsDir)
	userStore, err := multiStore.ForUser(uid)
	if err != nil {
		logger.Error("open user database", "user_id", uid, "error", err)
		os.Exit(1)
	}
	defer multiStore.Close()

	// Check if target already has data
	var entryCount int
	userStore.DB().QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&entryCount)
	if entryCount > 0 {
		logger.Warn("target database already has entries", "count", entryCount)
		fmt.Print("Continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			logger.Info("migration cancelled")
			os.Exit(0)
		}
	}

	logger.Info("migrating entries")
	migrateEntries(srcDB, userStore.DB(), logger)

	logger.Info("migrating rates")
	migrateRates(srcDB, userStore.DB(), logger)

	logger.Info("migrating invoices")
	migrateInvoices(srcDB, userStore.DB(), logger)

	logger.Info("migrating settings")
	migrateSettings(srcDB, userStore.DB(), logger)

	logger.Info("migration complete", "user_id", uid)
}

func migrateEntries(src, dst *sql.DB, logger *slog.Logger) {
	rows, err := src.Query(`SELECT date, category, hours, notes, created_at FROM entries ORDER BY id`)
	if err != nil {
		logger.Warn("no entries to migrate", "error", err)
		return
	}
	defer rows.Close()

	tx, err := dst.Begin()
	if err != nil {
		logger.Error("begin transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO entries (date, category, hours, notes, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		logger.Error("prepare statement", "error", err)
		return
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var date, category, notes, createdAt string
		var hours float64
		if err := rows.Scan(&date, &category, &hours, &notes, &createdAt); err != nil {
			logger.Error("scan entry row", "error", err)
			continue
		}
		if _, err := stmt.Exec(date, category, hours, notes, createdAt); err != nil {
			logger.Error("insert entry", "error", err)
			continue
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("commit entries", "error", err)
		return
	}
	logger.Info("migrated entries", "count", count)
}

func migrateRates(src, dst *sql.DB, logger *slog.Logger) {
	rows, err := src.Query(`SELECT category, start_date, end_date, rate, created_at FROM rates ORDER BY id`)
	if err != nil {
		logger.Warn("no rates to migrate", "error", err)
		return
	}
	defer rows.Close()

	tx, err := dst.Begin()
	if err != nil {
		logger.Error("begin transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO rates (category, start_date, end_date, rate, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		logger.Error("prepare statement", "error", err)
		return
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var category, startDate, endDate, createdAt string
		var rate float64
		if err := rows.Scan(&category, &startDate, &endDate, &rate, &createdAt); err != nil {
			logger.Error("scan rate row", "error", err)
			continue
		}
		if _, err := stmt.Exec(category, startDate, endDate, rate, createdAt); err != nil {
			logger.Error("insert rate", "error", err)
			continue
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("commit rates", "error", err)
		return
	}
	logger.Info("migrated rates", "count", count)
}

func migrateInvoices(src, dst *sql.DB, logger *slog.Logger) {
	rows, err := src.Query(`SELECT invoice_number, month, category, invoice_date, due_date, subtotal, total, business_name, business_address, bank_name, account_name, account_number, sort_code, payment_terms, created_at FROM invoices ORDER BY id`)
	if err != nil {
		logger.Warn("no invoices to migrate", "error", err)
		return
	}
	defer rows.Close()

	tx, err := dst.Begin()
	if err != nil {
		logger.Error("begin transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO invoices (invoice_number, month, category, invoice_date, due_date, subtotal, total, business_name, business_address, bank_name, account_name, account_number, sort_code, payment_terms, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		logger.Error("prepare invoice statement", "error", err)
		return
	}
	defer stmt.Close()

	lineStmt, err := tx.Prepare(`INSERT INTO invoice_lines (invoice_id, category, hours, rate, amount) VALUES ((SELECT id FROM invoices WHERE invoice_number = ?), ?, ?, ?, ?)`)
	if err != nil {
		logger.Error("prepare line statement", "error", err)
		return
	}
	defer lineStmt.Close()

	count := 0
	lineCount := 0
	for rows.Next() {
		var invNum, month, category, invDate, dueDate, bizName, bizAddr, bankName, acctName, acctNum, sortCode, payTerms, createdAt string
		var subtotal, total float64
		if err := rows.Scan(&invNum, &month, &category, &invDate, &dueDate, &subtotal, &total, &bizName, &bizAddr, &bankName, &acctName, &acctNum, &sortCode, &payTerms, &createdAt); err != nil {
			logger.Error("scan invoice row", "error", err)
			continue
		}
		// Check for duplicate invoice number
		var existing int
		dst.QueryRow(`SELECT COUNT(*) FROM invoices WHERE invoice_number = ?`, invNum).Scan(&existing)
		if existing > 0 {
			logger.Warn("skipping duplicate invoice", "invoice_number", invNum)
			continue
		}
		if _, err := stmt.Exec(invNum, month, category, invDate, dueDate, subtotal, total, bizName, bizAddr, bankName, acctName, acctNum, sortCode, payTerms, createdAt); err != nil {
			logger.Error("insert invoice", "error", err)
			continue
		}

		// Migrate invoice lines (original invoice ID is not preserved)
		lineRows, lErr := src.Query(`SELECT category, hours, rate, amount FROM invoice_lines WHERE invoice_id = (SELECT id FROM invoices WHERE invoice_number = ?)`, invNum)
		if lErr != nil {
			continue
		}
		for lineRows.Next() {
			var lineCat string
			var lineHours, lineRate, lineAmount float64
			if err := lineRows.Scan(&lineCat, &lineHours, &lineRate, &lineAmount); err != nil {
				continue
			}
			if _, err := lineStmt.Exec(invNum, lineCat, lineHours, lineRate, lineAmount); err != nil {
				logger.Error("insert invoice line", "error", err)
				continue
			}
			lineCount++
		}
		lineRows.Close()
		count++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("commit invoices", "error", err)
		return
	}
	logger.Info("migrated invoices", "count", count, "line_items", lineCount)
}

func migrateSettings(src, dst *sql.DB, logger *slog.Logger) {
	rows, err := src.Query(`SELECT key, value FROM settings WHERE key NOT IN ('schema_migrations_version') ORDER BY key`)
	if err != nil {
		logger.Warn("no settings to migrate", "error", err)
		return
	}
	defer rows.Close()

	tx, err := dst.Begin()
	if err != nil {
		logger.Error("begin transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`)
	if err != nil {
		logger.Error("prepare settings statement", "error", err)
		return
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			logger.Error("scan settings row", "error", err)
			continue
		}
		if _, err := stmt.Exec(key, value); err != nil {
			logger.Error("insert setting", "key", key, "error", err)
			continue
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("commit settings", "error", err)
		return
	}
	logger.Info("migrated settings", "count", count)
}
