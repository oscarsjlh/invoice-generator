package db

import (
	"fmt"
)

func (s *Store) ListEntries() ([]Entry, error) {
	return s.ListEntriesFiltered("", "", "")
}

func (s *Store) ListEntriesFiltered(year, month, rate string) ([]Entry, error) {
	query := `
		SELECT
			e.id,
			e.date,
			e.category,
			e.hours,
			COALESCE(e.notes, ''),
			NOT EXISTS (
				SELECT 1
				FROM rates r
				WHERE r.category = e.category
				  AND e.date >= r.start_date
				  AND (r.end_date = '' OR e.date <= r.end_date)
			) AS missing_rate
		FROM entries e
		WHERE 1=1
	`
	args := []any{}
	if year != "" {
		query += ` AND strftime('%Y', e.date) = ?`
		args = append(args, year)
	}
	if month != "" {
		query += ` AND strftime('%m', e.date) = ?`
		args = append(args, month)
	}
	if rate != "" {
		query += ` AND e.category = ?`
		args = append(args, rate)
	}
	query += ` ORDER BY e.date DESC, e.id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Date, &entry.Category, &entry.Hours, &entry.Notes, &entry.MissingRate); err != nil {
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

func (s *Store) GetEntry(id int64) (Entry, error) {
	var entry Entry
	err := s.db.QueryRow(
		`
		SELECT
			e.id,
			e.date,
			e.category,
			e.hours,
			COALESCE(e.notes, ''),
			NOT EXISTS (
				SELECT 1
				FROM rates r
				WHERE r.category = e.category
				  AND e.date >= r.start_date
				  AND (r.end_date = '' OR e.date <= r.end_date)
			) AS missing_rate
		FROM entries e
		WHERE e.id = ?`,
		id,
	).Scan(&entry.ID, &entry.Date, &entry.Category, &entry.Hours, &entry.Notes, &entry.MissingRate)
	if err != nil {
		return Entry{}, fmt.Errorf("get entry: %w", err)
	}
	return entry, nil
}

func (s *Store) UpdateEntry(id int64, date, category string, hours float64, notes string) error {
	_, err := s.db.Exec(
		`UPDATE entries SET date = ?, category = ?, hours = ?, notes = ? WHERE id = ?`,
		date, category, hours, notes, id,
	)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
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
