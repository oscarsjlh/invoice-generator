package db

import (
	"fmt"
	"strings"
)

func (s *Store) CreateOCRSession() (int64, error) {
	result, err := s.db.Exec(`INSERT INTO ocr_import_sessions (state) VALUES ('uploaded')`)
	if err != nil {
		return 0, fmt.Errorf("create ocr session: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) GetOCRSession(id int64) (OCRSession, error) {
	var sess OCRSession
	err := s.db.QueryRow(
		`SELECT id, state, COALESCE(error_msg,''), created_at, updated_at FROM ocr_import_sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.State, &sess.ErrorMsg, &sess.CreatedAt, &sess.UpdatedAt)
	if err != nil {
		return OCRSession{}, fmt.Errorf("get ocr session: %w", err)
	}
	return sess, nil
}

func (s *Store) UpdateOCRSessionState(id int64, state string, errorMsg string) error {
	_, err := s.db.Exec(
		`UPDATE ocr_import_sessions SET state = ?, error_msg = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		state, errorMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update ocr session state: %w", err)
	}
	return nil
}

func (s *Store) ListOCRSessions() ([]OCRSession, error) {
	rows, err := s.db.Query(
		`SELECT id, state, COALESCE(error_msg,''), created_at, updated_at FROM ocr_import_sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list ocr sessions: %w", err)
	}
	defer rows.Close()

	var sessions []OCRSession
	for rows.Next() {
		var sess OCRSession
		if err := rows.Scan(&sess.ID, &sess.State, &sess.ErrorMsg, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) AddSessionImage(sessionID int64, fileName, filePath string, fileSize int64) error {
	_, err := s.db.Exec(
		`INSERT INTO ocr_session_images (session_id, file_name, file_path, file_size) VALUES (?, ?, ?, ?)`,
		sessionID, fileName, filePath, fileSize,
	)
	if err != nil {
		return fmt.Errorf("add session image: %w", err)
	}
	return nil
}

func (s *Store) GetSessionImages(sessionID int64) ([]OCRSessionImage, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, file_name, file_path, file_size FROM ocr_session_images WHERE session_id = ? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get session images: %w", err)
	}
	defer rows.Close()

	var images []OCRSessionImage
	for rows.Next() {
		var img OCRSessionImage
		if err := rows.Scan(&img.ID, &img.SessionID, &img.FileName, &img.FilePath, &img.FileSize); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (s *Store) SaveDraftEntries(sessionID int64, drafts []OCRDraftEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM ocr_draft_entries WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("clear drafts: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO ocr_draft_entries
		(session_id, date_raw, date_normalized, category_raw, category_normalized,
		 hours_raw, hours_normalized, notes_raw, notes_normalized, confidence, needs_review, confirmed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, d := range drafts {
		if _, err := stmt.Exec(
			sessionID,
			d.DateRaw, d.DateNormalized,
			d.CategoryRaw, d.CategoryNormalized,
			d.HoursRaw, d.HoursNormalized,
			d.NotesRaw, d.NotesNormalized,
			d.Confidence, d.NeedsReview,
		); err != nil {
			return fmt.Errorf("insert draft: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) GetDraftEntries(sessionID int64) ([]OCRDraftEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, date_raw, date_normalized, category_raw, category_normalized,
		        hours_raw, hours_normalized, notes_raw, notes_normalized, confidence, needs_review, confirmed
		 FROM ocr_draft_entries WHERE session_id = ? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get draft entries: %w", err)
	}
	defer rows.Close()

	var drafts []OCRDraftEntry
	for rows.Next() {
		var d OCRDraftEntry
		err := rows.Scan(
			&d.ID, &d.SessionID,
			&d.DateRaw, &d.DateNormalized,
			&d.CategoryRaw, &d.CategoryNormalized,
			&d.HoursRaw, &d.HoursNormalized,
			&d.NotesRaw, &d.NotesNormalized,
			&d.Confidence, &d.NeedsReview, &d.Confirmed,
		)
		if err != nil {
			return nil, fmt.Errorf("scan draft: %w", err)
		}
		drafts = append(drafts, d)
	}
	return drafts, rows.Err()
}

func (s *Store) GetDraftEntry(id int64) (OCRDraftEntry, error) {
	var d OCRDraftEntry
	err := s.db.QueryRow(
		`SELECT id, session_id, date_raw, date_normalized, category_raw, category_normalized,
		        hours_raw, hours_normalized, notes_raw, notes_normalized, confidence, needs_review, confirmed
		 FROM ocr_draft_entries WHERE id = ?`, id,
	).Scan(
		&d.ID, &d.SessionID,
		&d.DateRaw, &d.DateNormalized,
		&d.CategoryRaw, &d.CategoryNormalized,
		&d.HoursRaw, &d.HoursNormalized,
		&d.NotesRaw, &d.NotesNormalized,
		&d.Confidence, &d.NeedsReview, &d.Confirmed,
	)
	if err != nil {
		return OCRDraftEntry{}, fmt.Errorf("get draft entry: %w", err)
	}
	return d, nil
}

func (s *Store) UpdateDraftEntry(id int64, date, category, hours, notes string) error {
	hoursVal := 0.0
	if hours != "" {
		if _, err := fmt.Sscanf(hours, "%f", &hoursVal); err != nil || hoursVal <= 0 {
			return fmt.Errorf("invalid hours value %q", hours)
		}
	}
	_, err := s.db.Exec(
		`UPDATE ocr_draft_entries SET date_normalized = ?, category_normalized = ?, hours_normalized = ?, notes_normalized = ? WHERE id = ?`,
		date, category, hoursVal, notes, id,
	)
	if err != nil {
		return fmt.Errorf("update draft %d: %w", id, err)
	}
	return nil
}

func (s *Store) ConfirmDraftEntries(sessionID int64, ids []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, id := range ids {
		var d OCRDraftEntry
		err := tx.QueryRow(
			`SELECT id, session_id, date_raw, date_normalized, category_raw, category_normalized,
			        hours_raw, hours_normalized, notes_raw, notes_normalized, confidence, needs_review, confirmed
			 FROM ocr_draft_entries WHERE id = ?`, id,
		).Scan(
			&d.ID, &d.SessionID,
			&d.DateRaw, &d.DateNormalized,
			&d.CategoryRaw, &d.CategoryNormalized,
			&d.HoursRaw, &d.HoursNormalized,
			&d.NotesRaw, &d.NotesNormalized,
			&d.Confidence, &d.NeedsReview, &d.Confirmed,
		)
		if err != nil {
			return fmt.Errorf("get draft %d: %w", id, err)
		}
		if d.SessionID != sessionID {
			return fmt.Errorf("draft %d does not belong to session %d", id, sessionID)
		}
		if d.DateNormalized == "" || d.CategoryNormalized == "" || d.HoursNormalized <= 0 {
			continue
		}

		if _, err := tx.Exec(
			`INSERT INTO entries (date, category, hours, notes) VALUES (?, ?, ?, ?)`,
			d.DateNormalized, d.CategoryNormalized, d.HoursNormalized, strings.TrimSpace(d.NotesNormalized),
		); err != nil {
			return fmt.Errorf("create entry from draft %d: %w", id, err)
		}

		if _, err := tx.Exec(`UPDATE ocr_draft_entries SET confirmed = 1 WHERE id = ?`, id); err != nil {
			return fmt.Errorf("mark confirmed %d: %w", id, err)
		}
	}

	var remaining int
	err = tx.QueryRow(
		`SELECT COUNT(*) FROM ocr_draft_entries WHERE session_id = ? AND confirmed = 0`,
		sessionID,
	).Scan(&remaining)
	if err != nil {
		return fmt.Errorf("count unconfirmed: %w", err)
	}
	var newState string
	if remaining == 0 {
		newState = "confirmed"
	} else {
		newState = "confirmed_part"
	}

	if _, err := tx.Exec(
		`UPDATE ocr_import_sessions SET state = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		newState, sessionID,
	); err != nil {
		return fmt.Errorf("update session state: %w", err)
	}

	return tx.Commit()
}

func (s *Store) countUnconfirmedDrafts(sessionID int64) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM ocr_draft_entries WHERE session_id = ? AND confirmed = 0`,
		sessionID,
	).Scan(&count)
	return count, err
}

func (s *Store) DeleteOCRSession(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM ocr_import_sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete ocr session: %w", err)
	}
	return nil
}

func (s *Store) CleanupStaleOCRSessions(maxAgeHours int) (int, error) {
	result, err := s.db.Exec(
		`DELETE FROM ocr_import_sessions WHERE state IN ('failed', 'confirmed') AND (strftime('%s','now') - strftime('%s',updated_at)) > ?`,
		maxAgeHours*3600,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup stale sessions: %w", err)
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}
