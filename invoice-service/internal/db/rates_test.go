package db

import (
	"testing"
	"time"
)

func TestRateCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	err := store.CreateRate("Consulting", "2024-01-01", "", 150.00)
	if err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	rate, err := store.GetRate(1)
	if err != nil {
		t.Fatalf("GetRate: %v", err)
	}
	if rate.Category != "Consulting" || rate.Rate != 150.00 {
		t.Errorf("GetRate returned wrong values: %+v", rate)
	}
}

func TestListRates(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate 1: %v", err)
	}
	if err := store.CreateRate("Design", "2024-01-01", "", 120.00); err != nil {
		t.Fatalf("CreateRate 2: %v", err)
	}

	rates, err := store.ListRates()
	if err != nil {
		t.Fatalf("ListRates: %v", err)
	}
	if len(rates) != 2 {
		t.Fatalf("expected 2 rates, got %d", len(rates))
	}
	if rates[0].Category != "Consulting" || rates[1].Category != "Design" {
		t.Errorf("rates not sorted correctly: %+v", rates)
	}
}

func TestListActiveRates(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	if err := store.CreateRate("Active Open", yesterday, "", 150.00); err != nil {
		t.Fatalf("CreateRate active open: %v", err)
	}
	if err := store.CreateRate("Active Ends Today", yesterday, today, 125.00); err != nil {
		t.Fatalf("CreateRate active ends today: %v", err)
	}
	if err := store.CreateRate("Expired", yesterday, yesterday, 100.00); err != nil {
		t.Fatalf("CreateRate expired: %v", err)
	}
	if err := store.CreateRate("Future", tomorrow, "", 175.00); err != nil {
		t.Fatalf("CreateRate future: %v", err)
	}

	rates, err := store.ListActiveRates(today)
	if err != nil {
		t.Fatalf("ListActiveRates: %v", err)
	}
	if len(rates) != 2 {
		t.Fatalf("expected 2 active rates, got %d: %+v", len(rates), rates)
	}
	categories := map[string]bool{}
	for _, rate := range rates {
		categories[rate.Category] = true
	}
	if !categories["Active Open"] || !categories["Active Ends Today"] {
		t.Fatalf("expected active rates only, got %+v", rates)
	}
	if categories["Expired"] || categories["Future"] {
		t.Fatalf("inactive rates were returned: %+v", rates)
	}
}

func TestUpdateRate(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	err := store.UpdateRate(1, "Consulting", "2024-01-01", "", 175.00)
	if err != nil {
		t.Fatalf("UpdateRate: %v", err)
	}

	rate, err := store.GetRate(1)
	if err != nil {
		t.Fatalf("GetRate: %v", err)
	}
	if rate.Rate != 175.00 {
		t.Errorf("rate = %v, want 175.00", rate.Rate)
	}
}

func TestDeleteRate(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateRate("Consulting", "2024-01-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	err := store.DeleteRate(1)
	if err != nil {
		t.Fatalf("DeleteRate: %v", err)
	}

	_, err = store.GetRate(1)
	if err == nil {
		t.Fatal("expected error for deleted rate, got nil")
	}

	rates, err := store.ListRates()
	if err != nil {
		t.Fatalf("ListRates: %v", err)
	}
	if len(rates) != 0 {
		t.Errorf("expected 0 rates after delete, got %d", len(rates))
	}
}

func TestListCategories(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := store.CreateRate("Design", "2024-01-01", "", 120.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	cats, err := store.ListCategories()
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	if cats[0] != "Consulting" || cats[1] != "Design" {
		t.Errorf("categories not sorted: %+v", cats)
	}
}

func TestListAvailableYears(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2023-11-10", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}

	years, err := store.ListAvailableYears()
	if err != nil {
		t.Fatalf("ListAvailableYears: %v", err)
	}
	if len(years) != 2 || years[0] != "2024" || years[1] != "2023" {
		t.Errorf("years = %v, want [2024 2023]", years)
	}
}

func TestListAvailableMonths(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-01-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-03-10", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateEntry("2023-11-10", "Other", 1.0, ""); err != nil {
		t.Fatalf("CreateEntry 3: %v", err)
	}

	months, err := store.ListAvailableMonths("2024")
	if err != nil {
		t.Fatalf("ListAvailableMonths: %v", err)
	}
	if len(months) != 2 || months[0] != "01" || months[1] != "03" {
		t.Errorf("months = %v, want [01 03]", months)
	}

	// Filter by non-existent year
	noMonths, err := store.ListAvailableMonths("2099")
	if err != nil {
		t.Fatalf("ListAvailableMonths empty: %v", err)
	}
	if len(noMonths) != 0 {
		t.Errorf("expected 0 months for 2099, got %d", len(noMonths))
	}
}
