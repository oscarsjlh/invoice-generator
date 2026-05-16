package handler

import (
	"fmt"
	"net/http"
	"strings"
)

func (a *App) ratesPage(w http.ResponseWriter, r *http.Request) {
	rates, err := a.store.ListRates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPage(w, http.StatusOK, "rates.html", RatesPageData{
		Rates:  rates,
		Notice: noticeFromRequest(r),
	}, "rates_table.html")
}

func (a *App) createRate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	startDate, err := validateDate(r.FormValue("start_date"))
	if err != nil {
		http.Error(w, "start date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	endDate, err := validateDate(r.FormValue("end_date"))
	if err != nil {
		http.Error(w, "end date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if endDate < startDate {
		http.Error(w, "end date must be on or after start date", http.StatusBadRequest)
		return
	}
	rateValue, err := parsePositiveFloat(r.FormValue("rate"))
	if err != nil {
		http.Error(w, "rate must be a positive number", http.StatusBadRequest)
		return
	}

	if err := a.store.CreateRate(category, startDate, endDate, rateValue); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.redirect(w, r, "/rates", "Rate added")
}

func (a *App) deleteRate(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid rate id", http.StatusBadRequest)
		return
	}
	if err := a.store.DeleteRate(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isHTMX(r) {
		a.redirect(w, r, "/rates", "Rate deleted")
		return
	}

	rates, err := a.store.ListRates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "rates_table", RatesPageData{Rates: rates, Notice: fmt.Sprintf("Rate %d deleted", id)}, "rates_table.html")
}

func (a *App) editRateForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid rate id", http.StatusBadRequest)
		return
	}
	rate, err := a.store.GetRate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "rate_edit_row", rate, "rate_edit_row.html")
}

func (a *App) updateRate(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid rate id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	startDate, err := validateDate(r.FormValue("start_date"))
	if err != nil {
		http.Error(w, "start date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	endDate, err := validateDate(r.FormValue("end_date"))
	if err != nil {
		http.Error(w, "end date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if endDate < startDate {
		http.Error(w, "end date must be on or after start date", http.StatusBadRequest)
		return
	}
	rateValue, err := parsePositiveFloat(r.FormValue("rate"))
	if err != nil {
		http.Error(w, "rate must be a positive number", http.StatusBadRequest)
		return
	}

	if err := a.store.UpdateRate(id, category, startDate, endDate, rateValue); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rates, err := a.store.ListRates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "rates_table", RatesPageData{Rates: rates, Notice: fmt.Sprintf("Rate %d updated", id)}, "rates_table.html")
}

func (a *App) ratesTable(w http.ResponseWriter, r *http.Request) {
	rates, err := a.store.ListRates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "rates_table", RatesPageData{Rates: rates}, "rates_table.html")
}
