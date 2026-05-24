package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply pragma %q for database %s: %w", pragma, path, err)
		}
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(dir string) error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)

		if alreadyApplied, err := s.isMigrationApplied(filename); err != nil {
			return fmt.Errorf("check migration status %s: %w", filename, err)
		} else if alreadyApplied {
			continue
		}

		contents, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		statements := splitStatements(string(contents))
		for i, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("run migration %s statement %d: %w", filename, i+1, err)
			}
		}

		if err := s.recordMigration(filename); err != nil {
			return fmt.Errorf("record migration %s: %w", filename, err)
		}
	}

	return nil
}

func (s *Store) isMigrationApplied(filename string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = ?`, filename).Scan(&count)
	if err != nil {
		// Table doesn't exist yet — no migrations have been applied
		return false, nil
	}
	return count > 0, nil
}

func (s *Store) recordMigration(filename string) error {
	_, err := s.db.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, filename)
	return err
}

func splitStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(content); i++ {
		c := content[i]

		if inQuote {
			current.WriteByte(c)
			if c == quoteChar {
				inQuote = false
			}
			continue
		}

		switch c {
		case '\'', '"':
			inQuote = true
			quoteChar = c
			current.WriteByte(c)
		case ';':
			statements = append(statements, current.String())
			current.Reset()
		default:
			current.WriteByte(c)
		}
	}

	if rest := strings.TrimSpace(current.String()); rest != "" {
		statements = append(statements, rest)
	}

	return statements
}

func (s *Store) DB() *sql.DB {
	return s.db
}
