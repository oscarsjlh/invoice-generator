package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
)

func main() {
	entriesPath := flag.String("entries", "", "path to entries CSV")
	ratesPath := flag.String("rates", "", "path to rates CSV")
	flag.Parse()

	if *entriesPath == "" || *ratesPath == "" {
		log.Fatal("usage: go run ./cmd/migrate-csv --entries entries.csv --rates rates.csv")
	}

	cfg := config.Load()
	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(cfg.MigrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	entriesImported, err := importEntries(store, *entriesPath)
	if err != nil {
		log.Fatalf("import entries: %v", err)
	}
	ratesImported, err := importRates(store, *ratesPath)
	if err != nil {
		log.Fatalf("import rates: %v", err)
	}

	log.Printf("imported %d entries and %d rates\n", entriesImported, ratesImported)
}

func importEntries(store *db.Store, path string) (int, error) {
	records, err := readCSV(path)
	if err != nil {
		return 0, err
	}
	count := 0
	for i, record := range records {
		if i == 0 && looksLikeHeader(record) {
			continue
		}
		if len(record) < 3 {
			continue
		}
		date, err := normalizeDate(record[0])
		if err != nil {
			return count, fmt.Errorf("entries row %d date: %w", i+1, err)
		}
		category := strings.TrimSpace(record[1])
		if category == "" {
			continue
		}
		hours, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		if err != nil || hours <= 0 {
			return count, fmt.Errorf("entries row %d hours must be positive", i+1)
		}
		notes := ""
		if len(record) > 3 {
			notes = strings.TrimSpace(record[3])
		}
		if err := store.CreateEntry(date, category, hours, notes); err != nil {
			return count, fmt.Errorf("entries row %d insert: %w", i+1, err)
		}
		count++
	}
	return count, nil
}

func importRates(store *db.Store, path string) (int, error) {
	records, err := readCSV(path)
	if err != nil {
		return 0, err
	}
	count := 0
	for i, record := range records {
		if i == 0 && looksLikeHeader(record) {
			continue
		}
		if len(record) < 4 {
			continue
		}
		category := strings.TrimSpace(record[0])
		if category == "" {
			continue
		}
		startDate, err := normalizeDate(record[1])
		if err != nil {
			return count, fmt.Errorf("rates row %d start date: %w", i+1, err)
		}
		endDate, err := normalizeDate(record[2])
		if err != nil {
			return count, fmt.Errorf("rates row %d end date: %w", i+1, err)
		}
		rateValue, err := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)
		if err != nil || rateValue <= 0 {
			return count, fmt.Errorf("rates row %d rate must be positive", i+1)
		}
		if err := store.CreateRate(category, startDate, endDate, rateValue); err != nil {
			return count, fmt.Errorf("rates row %d insert: %w", i+1, err)
		}
		count++
	}
	return count, nil
}

func readCSV(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv %s: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

func normalizeDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty date")
	}

	if serial, err := strconv.Atoi(value); err == nil {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		return base.AddDate(0, 0, serial).Format("2006-01-02"), nil
	}

	formats := []string{"2006-01-02", "02/01/2006", "2/1/2006", "02/01/06", "2/1/06"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unsupported date format %q", value)
}

func looksLikeHeader(record []string) bool {
	joined := strings.ToLower(strings.Join(record, ","))
	return strings.Contains(joined, "date") || strings.Contains(joined, "category") || strings.Contains(joined, "rate")
}
