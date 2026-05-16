package handler

import (
	"net/http"
	"strings"

	"invoice-app/internal/db"
)

func (a *App) settingsPage(w http.ResponseWriter, r *http.Request) {
	settings, err := a.store.LoadSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPage(w, http.StatusOK, "settings.html", SettingsPageData{
		Settings: settings,
		Notice:   noticeFromRequest(r),
	}, "settings.html")
}

func (a *App) saveSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dueDays, err := parsePositiveInt(r.FormValue("default_due_days"))
	if err != nil {
		http.Error(w, "default due days must be a positive integer", http.StatusBadRequest)
		return
	}

	settings := db.Settings{
		BusinessName:       strings.TrimSpace(r.FormValue("business_name")),
		BusinessAddress:    strings.TrimSpace(r.FormValue("business_address")),
		BankName:           strings.TrimSpace(r.FormValue("bank_name")),
		AccountName:        strings.TrimSpace(r.FormValue("account_name")),
		AccountNumber:      strings.TrimSpace(r.FormValue("account_number")),
		SortCode:           strings.TrimSpace(r.FormValue("sort_code")),
		PaymentTerms:       strings.TrimSpace(r.FormValue("payment_terms")),
		DefaultDueDays:     dueDays,
		CustomerName:       strings.TrimSpace(r.FormValue("customer_name")),
		CustomerTitle:      strings.TrimSpace(r.FormValue("customer_title")),
		CustomerEmail:      strings.TrimSpace(r.FormValue("customer_email")),
		CustomerAddress:    strings.TrimSpace(r.FormValue("customer_address")),
		CustomerPostalCode: strings.TrimSpace(r.FormValue("customer_postal_code")),
		CustomerCity:       strings.TrimSpace(r.FormValue("customer_city")),
	}
	if err := a.store.SaveSettings(settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.redirect(w, r, "/settings", "Settings saved")
}
