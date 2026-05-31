package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var nonAlphaNumeric = regexp.MustCompile(`[^A-Z0-9]+`)

func (s *Store) ListMonthlySummary() ([]MonthlySummary, error) {
	rows, err := s.db.Query(`
		SELECT
			e.category,
			strftime('%Y-%m', e.date) AS month,
			ROUND(SUM(e.hours), 2) AS total_hours,
			ROUND(SUM(e.hours * r.rate), 2) AS total_amount
		FROM entries e
		JOIN rates r
			ON e.category = r.category
			AND e.date >= r.start_date
			AND (r.end_date = '' OR e.date <= r.end_date)
		GROUP BY e.category, month
		ORDER BY month DESC, e.category ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list monthly summary: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var summary []MonthlySummary
	for rows.Next() {
		var row MonthlySummary
		if err := rows.Scan(&row.Category, &row.Month, &row.TotalHours, &row.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		summary = append(summary, row)
	}

	return summary, rows.Err()
}

func (s *Store) ListAvailableYears() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT strftime('%Y', date) FROM entries ORDER BY 1 DESC`)
	if err != nil {
		return nil, fmt.Errorf("list available years: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var years []string
	for rows.Next() {
		var y string
		if err := rows.Scan(&y); err != nil {
			return nil, fmt.Errorf("scan year: %w", err)
		}
		years = append(years, y)
	}
	return years, rows.Err()
}

func (s *Store) ListAvailableMonths(year string) ([]string, error) {
	query := `SELECT DISTINCT strftime('%m', date) FROM entries`
	args := []any{}
	if year != "" {
		query += ` WHERE strftime('%Y', date) = ?`
		args = append(args, year)
	}
	query += ` ORDER BY 1 ASC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list available months: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var months []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("scan month: %w", err)
		}
		months = append(months, m)
	}
	return months, rows.Err()
}

func (s *Store) FilteredSummary(year, month string) ([]MonthlySummary, float64, float64, error) {
	query := `
		SELECT
			e.category,
			ROUND(SUM(e.hours), 2) AS total_hours,
			ROUND(SUM(e.hours * r.rate), 2) AS total_amount
		FROM entries e
		JOIN rates r
			ON e.category = r.category
			AND e.date >= r.start_date
			AND (r.end_date = '' OR e.date <= r.end_date)
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
	query += ` GROUP BY e.category ORDER BY e.category ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("filtered summary: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var summary []MonthlySummary
	var totalHours, totalAmount float64
	for rows.Next() {
		var row MonthlySummary
		if err := rows.Scan(&row.Category, &row.TotalHours, &row.TotalAmount); err != nil {
			return nil, 0, 0, fmt.Errorf("scan filtered summary: %w", err)
		}
		totalHours += row.TotalHours
		totalAmount += row.TotalAmount
		summary = append(summary, row)
	}

	return summary, totalHours, totalAmount, rows.Err()
}

func (s *Store) ListInvoices() ([]InvoiceSummary, error) {
	rows, err := s.db.Query(`
		SELECT id, invoice_number, month, category, total, invoice_date, due_date, created_at
		FROM invoices
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var invoices []InvoiceSummary
	for rows.Next() {
		var invoice InvoiceSummary
		if err := rows.Scan(
			&invoice.ID,
			&invoice.InvoiceNumber,
			&invoice.Month,
			&invoice.Category,
			&invoice.Total,
			&invoice.InvoiceDate,
			&invoice.DueDate,
			&invoice.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invoice summary: %w", err)
		}
		invoices = append(invoices, invoice)
	}

	return invoices, rows.Err()
}

func (s *Store) CountUnratedEntries(month, category string) (int, error) {
	row := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM entries e
		WHERE strftime('%Y-%m', e.date) = ?
		  AND (? = 'All' OR e.category = ?)
		  AND NOT EXISTS (
			SELECT 1
			FROM rates r
			WHERE r.category = e.category
			  AND e.date >= r.start_date
			  AND (r.end_date = '' OR e.date <= r.end_date)
		  )
	`, month, category, category)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count unrated entries: %w", err)
	}
	return count, nil
}

func (s *Store) CountUnratedEntriesFiltered(year, month string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM entries e
		WHERE NOT EXISTS (
			SELECT 1
			FROM rates r
			WHERE r.category = e.category
			  AND e.date >= r.start_date
			  AND (r.end_date = '' OR e.date <= r.end_date)
		)
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

	var count int
	if err := s.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count filtered unrated entries: %w", err)
	}
	return count, nil
}

func (s *Store) GenerateInvoice(month, category, invoiceDate string, dueDays int, invoiceNumber string, settings Settings) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin invoice transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	unrated, err := countUnratedEntriesTx(tx, month, category)
	if err != nil {
		return 0, err
	}
	if unrated > 0 {
		return 0, fmt.Errorf("%d entries in %s have no matching rate", unrated, month)
	}

	parsedInvoiceDate, err := time.Parse("2006-01-02", invoiceDate)
	if err != nil {
		return 0, fmt.Errorf("parse invoice date: %w", err)
	}
	dueDate := parsedInvoiceDate.AddDate(0, 0, dueDays).Format("2006-01-02")

	lines, total, err := queryInvoiceLinesTx(tx, month, category)
	if err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no invoiceable entries found for %s", month)
	}

	invoiceNumber = strings.TrimSpace(invoiceNumber)
	if invoiceNumber == "" {
		invoiceNumber, err = nextInvoiceNumber(tx, month, category)
		if err != nil {
			return 0, err
		}
	} else if exists, err := invoiceNumberExists(tx, invoiceNumber); err != nil {
		return 0, err
	} else if exists {
		return 0, fmt.Errorf("invoice number %q already exists", invoiceNumber)
	}

	result, err := tx.Exec(`
		INSERT INTO invoices (
			invoice_number,
			month,
			category,
			invoice_date,
			due_date,
			subtotal,
			total,
			business_name,
			business_address,
			bank_name,
			account_name,
			account_number,
			sort_code,
			business_utr,
			show_payment_due,
			payment_terms,
			customer_name,
			customer_title,
			customer_email,
			customer_address,
			customer_postal_code,
			customer_city
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		invoiceNumber,
		month,
		category,
		invoiceDate,
		dueDate,
		total,
		total,
		settings.BusinessName,
		settings.BusinessAddress,
		settings.BankName,
		settings.AccountName,
		settings.AccountNumber,
		settings.SortCode,
		settings.UTR,
		settings.ShowPaymentDue,
		settings.PaymentTerms,
		settings.CustomerName,
		settings.CustomerTitle,
		settings.CustomerEmail,
		settings.CustomerAddress,
		settings.CustomerPostalCode,
		settings.CustomerCity,
	)
	if err != nil {
		return 0, fmt.Errorf("insert invoice: %w", err)
	}

	invoiceID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("invoice last insert id: %w", err)
	}

	for _, line := range lines {
		if _, err := tx.Exec(`
			INSERT INTO invoice_lines (invoice_id, category, hours, rate, amount, service_dates)
			VALUES (?, ?, ?, ?, ?, ?)
		`, invoiceID, line.Category, line.Hours, line.Rate, line.Amount, strings.Join(line.ServiceDates, ",")); err != nil {
			return 0, fmt.Errorf("insert invoice line: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit invoice transaction: %w", err)
	}

	return invoiceID, nil
}

func countUnratedEntriesTx(tx *sql.Tx, month, category string) (int, error) {
	row := tx.QueryRow(`
		SELECT COUNT(*)
		FROM entries e
		WHERE strftime('%Y-%m', e.date) = ?
		  AND (? = 'All' OR e.category = ?)
		  AND NOT EXISTS (
			SELECT 1
			FROM rates r
			WHERE r.category = e.category
			  AND e.date >= r.start_date
			  AND (r.end_date = '' OR e.date <= r.end_date)
		  )
	`, month, category, category)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count unrated entries: %w", err)
	}
	return count, nil
}

func queryInvoiceLinesTx(tx *sql.Tx, month, category string) ([]InvoiceLine, float64, error) {
	rows, err := tx.Query(`
		SELECT
			e.category,
			ROUND(SUM(e.hours), 2) AS hours,
			r.rate,
			ROUND(SUM(e.hours * r.rate), 2) AS amount,
			GROUP_CONCAT(e.date, ',') AS service_dates
		FROM entries e
		JOIN rates r
			ON e.category = r.category
			AND e.date >= r.start_date
			AND (r.end_date = '' OR e.date <= r.end_date)
		WHERE strftime('%Y-%m', e.date) = ?
		  AND (? = 'All' OR e.category = ?)
		GROUP BY e.category, r.rate
		ORDER BY e.category ASC
	`, month, category, category)
	if err != nil {
		return nil, 0, fmt.Errorf("query invoice lines: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	lines := []InvoiceLine{}
	var total float64
	for rows.Next() {
		var line InvoiceLine
		var serviceDates string
		if err := rows.Scan(&line.Category, &line.Hours, &line.Rate, &line.Amount, &serviceDates); err != nil {
			return nil, 0, fmt.Errorf("scan invoice line: %w", err)
		}
		line.ServiceDates = splitServiceDates(serviceDates)
		total += line.Amount
		lines = append(lines, line)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return lines, total, nil
}

func (s *Store) GetInvoice(id int64) (Invoice, error) {
	var invoice Invoice
	var showPaymentDue int
	row := s.db.QueryRow(`
		SELECT
			id,
			invoice_number,
			month,
			category,
			invoice_date,
			due_date,
			subtotal,
			total,
			business_name,
			business_address,
			bank_name,
			account_name,
			account_number,
			sort_code,
			business_utr,
			show_payment_due,
			payment_terms,
			customer_name,
			customer_title,
			customer_email,
			customer_address,
			customer_postal_code,
			customer_city
		FROM invoices
		WHERE id = ?
	`, id)

	if err := row.Scan(
		&invoice.ID,
		&invoice.InvoiceNumber,
		&invoice.Month,
		&invoice.Category,
		&invoice.InvoiceDate,
		&invoice.DueDate,
		&invoice.Subtotal,
		&invoice.Total,
		&invoice.BusinessName,
		&invoice.BusinessAddress,
		&invoice.BankName,
		&invoice.AccountName,
		&invoice.AccountNumber,
		&invoice.SortCode,
		&invoice.UTR,
		&showPaymentDue,
		&invoice.PaymentTerms,
		&invoice.CustomerName,
		&invoice.CustomerTitle,
		&invoice.CustomerEmail,
		&invoice.CustomerAddress,
		&invoice.CustomerPostalCode,
		&invoice.CustomerCity,
	); err != nil {
		if err == sql.ErrNoRows {
			return Invoice{}, err
		}
		return Invoice{}, fmt.Errorf("get invoice: %w", err)
	}
	invoice.ShowPaymentDue = showPaymentDue != 0

	rows, err := s.db.Query(`
		SELECT id, category, hours, rate, amount, service_dates
		FROM invoice_lines
		WHERE invoice_id = ?
		ORDER BY category ASC, id ASC
	`, id)
	if err != nil {
		return Invoice{}, fmt.Errorf("list invoice lines: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var line InvoiceLine
		var serviceDates string
		if err := rows.Scan(&line.ID, &line.Category, &line.Hours, &line.Rate, &line.Amount, &serviceDates); err != nil {
			return Invoice{}, fmt.Errorf("scan invoice line: %w", err)
		}
		line.ServiceDates = splitServiceDates(serviceDates)
		invoice.Lines = append(invoice.Lines, line)
	}

	return invoice, rows.Err()
}

func (s *Store) DeleteInvoice(id int64) error {
	result, err := s.db.Exec(`DELETE FROM invoices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete invoice rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func nextInvoiceNumber(tx *sql.Tx, month, category string) (string, error) {
	base := fmt.Sprintf("INV-%s-%s", month, invoiceSlug(category))
	for suffix := 1; suffix <= 9999; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}

		var exists int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM invoices WHERE invoice_number = ?`, candidate).Scan(&exists); err != nil {
			return "", fmt.Errorf("check invoice number: %w", err)
		}
		if exists == 0 {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not allocate invoice number for %s/%s", month, category)
}

func invoiceNumberExists(tx *sql.Tx, invoiceNumber string) (bool, error) {
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM invoices WHERE invoice_number = ?`, invoiceNumber).Scan(&exists); err != nil {
		return false, fmt.Errorf("check invoice number: %w", err)
	}
	return exists > 0, nil
}

func splitServiceDates(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	dates := strings.Split(value, ",")
	clean := dates[:0]
	seen := map[string]bool{}
	for _, date := range dates {
		date = strings.TrimSpace(date)
		if date == "" || seen[date] {
			continue
		}
		seen[date] = true
		clean = append(clean, date)
	}
	sort.Strings(clean)
	return clean
}

func invoiceSlug(category string) string {
	if strings.TrimSpace(category) == "" || category == "All" {
		return "ALL"
	}
	slug := strings.ToUpper(strings.TrimSpace(category))
	slug = nonAlphaNumeric.ReplaceAllString(slug, "")
	if slug == "" {
		return "ALL"
	}
	return slug
}
