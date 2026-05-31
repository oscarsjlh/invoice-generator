package db

import "fmt"

func (s *Store) ListRates() ([]Rate, error) {
	rows, err := s.db.Query(`
		SELECT id, category, start_date, end_date, rate
		FROM rates
		ORDER BY category ASC, start_date DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list rates: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rates := []Rate{}
	for rows.Next() {
		var rate Rate
		if err := rows.Scan(&rate.ID, &rate.Category, &rate.StartDate, &rate.EndDate, &rate.Rate); err != nil {
			return nil, fmt.Errorf("scan rate: %w", err)
		}
		rates = append(rates, rate)
	}

	return rates, rows.Err()
}

func (s *Store) ListActiveRates(today string) ([]Rate, error) {
	rows, err := s.db.Query(`
		SELECT id, category, start_date, end_date, rate
		FROM rates
		WHERE start_date <= ?
		  AND (end_date = '' OR end_date >= ?)
		ORDER BY category ASC, start_date DESC, id DESC
	`, today, today)
	if err != nil {
		return nil, fmt.Errorf("list active rates: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rates := []Rate{}
	for rows.Next() {
		var rate Rate
		if err := rows.Scan(&rate.ID, &rate.Category, &rate.StartDate, &rate.EndDate, &rate.Rate); err != nil {
			return nil, fmt.Errorf("scan active rate: %w", err)
		}
		rates = append(rates, rate)
	}

	return rates, rows.Err()
}

func (s *Store) CreateRate(category, startDate, endDate string, rate float64) error {
	if err := s.checkRateOverlap(category, startDate, endDate, 0); err != nil {
		return err
	}

	_, err := s.db.Exec(
		`INSERT INTO rates (category, start_date, end_date, rate) VALUES (?, ?, ?, ?)`,
		category,
		startDate,
		endDate,
		rate,
	)
	if err != nil {
		return fmt.Errorf("create rate: %w", err)
	}
	return nil
}

func (s *Store) DeleteRate(id int64) error {
	_, err := s.db.Exec(`DELETE FROM rates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete rate: %w", err)
	}
	return nil
}

func (s *Store) GetRate(id int64) (Rate, error) {
	var rate Rate
	err := s.db.QueryRow(
		`SELECT id, category, start_date, end_date, rate FROM rates WHERE id = ?`, id,
	).Scan(&rate.ID, &rate.Category, &rate.StartDate, &rate.EndDate, &rate.Rate)
	if err != nil {
		return Rate{}, fmt.Errorf("get rate: %w", err)
	}
	return rate, nil
}

func (s *Store) UpdateRate(id int64, category, startDate, endDate string, rate float64) error {
	if err := s.checkRateOverlap(category, startDate, endDate, id); err != nil {
		return err
	}

	_, err := s.db.Exec(
		`UPDATE rates SET category = ?, start_date = ?, end_date = ?, rate = ? WHERE id = ?`,
		category, startDate, endDate, rate, id,
	)
	if err != nil {
		return fmt.Errorf("update rate: %w", err)
	}
	return nil
}

func (s *Store) checkRateOverlap(category, startDate, endDate string, excludeID int64) error {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM rates
		WHERE category = ?
		  AND id != ?
		  AND (end_date = '' OR end_date >= ?)
		  AND (? = '' OR start_date <= ?)
	`, category, excludeID, startDate, endDate, endDate).Scan(&count)
	if err != nil {
		return fmt.Errorf("check rate overlap: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("a rate for %q covering that date range already exists", category)
	}
	return nil
}

func (s *Store) ListCategories() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT category FROM (
			SELECT DISTINCT category FROM entries
			UNION
			SELECT DISTINCT category FROM rates
		)
		WHERE category <> ''
		ORDER BY category ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}
