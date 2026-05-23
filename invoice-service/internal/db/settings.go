package db

import "fmt"

func (s *Store) LoadSettings() (Settings, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return Settings{}, fmt.Errorf("load settings: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	settings := Settings{DefaultDueDays: 30}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return Settings{}, fmt.Errorf("scan setting: %w", err)
		}
		switch key {
		case "business_name":
			settings.BusinessName = value
		case "business_address":
			settings.BusinessAddress = value
		case "bank_name":
			settings.BankName = value
		case "account_name":
			settings.AccountName = value
		case "account_number":
			settings.AccountNumber = value
		case "sort_code":
			settings.SortCode = value
		case "payment_terms":
			settings.PaymentTerms = value
		case "default_due_days":
			if value == "" {
				continue
			}
			var dueDays int
			if _, err := fmt.Sscanf(value, "%d", &dueDays); err == nil && dueDays > 0 {
				settings.DefaultDueDays = dueDays
			}
		case "customer_name":
			settings.CustomerName = value
		case "customer_title":
			settings.CustomerTitle = value
		case "customer_email":
			settings.CustomerEmail = value
		case "customer_address":
			settings.CustomerAddress = value
		case "customer_postal_code":
			settings.CustomerPostalCode = value
		case "customer_city":
			settings.CustomerCity = value
		}
	}

	return settings, rows.Err()
}

func (s *Store) SaveSettings(settings Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin save settings: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	values := map[string]string{
		"business_name":        settings.BusinessName,
		"business_address":     settings.BusinessAddress,
		"bank_name":            settings.BankName,
		"account_name":         settings.AccountName,
		"account_number":       settings.AccountNumber,
		"sort_code":            settings.SortCode,
		"payment_terms":        settings.PaymentTerms,
		"default_due_days":     fmt.Sprintf("%d", settings.DefaultDueDays),
		"customer_name":        settings.CustomerName,
		"customer_title":       settings.CustomerTitle,
		"customer_email":       settings.CustomerEmail,
		"customer_address":     settings.CustomerAddress,
		"customer_postal_code": settings.CustomerPostalCode,
		"customer_city":        settings.CustomerCity,
	}

	for key, value := range values {
		if _, err := tx.Exec(`
			INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, key, value); err != nil {
			return fmt.Errorf("save setting %s: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settings: %w", err)
	}
	return nil
}
