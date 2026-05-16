package db

import (
	"fmt"
)

func (s *Store) ListEntries() ([]Entry, error) {
	rows, err := s.db.Query(`
		SELECT id, date, category, hours, COALESCE(notes, '')
		FROM entries
		ORDER BY date DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Date, &entry.Category, &entry.Hours, &entry.Notes); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (s *Store) CreateEntry(date, category string, hours float64, notes string) error {
	_, err := s.db.Exec(
		`INSERT INTO entries (date, category, hours, notes) VALUES (?, ?, ?, ?)`,
		date,
		category,
		hours,
		notes,
	)
	if err != nil {
		return fmt.Errorf("create entry: %w", err)
	}
	return nil
}

func (s *Store) DeleteEntry(id int64) error {
	_, err := s.db.Exec(`DELETE FROM entries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	return nil
}
