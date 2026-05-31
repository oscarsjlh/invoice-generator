package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"invoice-app/internal/db"
)

type RateHandlers struct {
	renderer *Renderer
}

func NewRateHandlers(renderer *Renderer) *RateHandlers {
	return &RateHandlers{renderer: renderer}
}

func (h *RateHandlers) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /rates", h.ratesPage)
	mux.HandleFunc("GET /rates/table", h.ratesTable)
	mux.HandleFunc("POST /rates", h.createRate)
	mux.HandleFunc("POST /rates/{id}/delete", h.deleteRate)
	mux.HandleFunc("GET /rates/{id}/edit", h.editRateForm)
	mux.HandleFunc("POST /rates/{id}", h.updateRate)
}

func (h *RateHandlers) ratesPage(w http.ResponseWriter, r *http.Request) {
	activeOnly := activeRatesOnly(r)
	rates, err := listRates(r, activeOnly)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list rates", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Page(w, r, http.StatusOK, "rates.html", RatesPageData{
		Rates:      rates,
		Notice:     noticeFromRequest(r),
		ActiveOnly: activeOnly,
	}, "rates_table.html")
}

func (h *RateHandlers) createRate(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		LoggerFromContext(r.Context()).Warn("create rate: missing category")
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	startDate, err := validateDate(r.FormValue("start_date"))
	if err != nil {
		LoggerFromContext(r.Context()).Warn("create rate: invalid start date", "value", r.FormValue("start_date"))
		http.Error(w, "start date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	endDate := strings.TrimSpace(r.FormValue("end_date"))
	if endDate != "" {
		var err error
		endDate, err = validateDate(endDate)
		if err != nil {
			LoggerFromContext(r.Context()).Warn("create rate: invalid end date", "value", r.FormValue("end_date"))
			http.Error(w, "end date must use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}
	if endDate != "" && endDate < startDate {
		LoggerFromContext(r.Context()).Warn("create rate: end before start", "start_date", startDate, "end_date", endDate)
		http.Error(w, "end date must be on or after start date", http.StatusBadRequest)
		return
	}
	rateValue, err := parsePositiveFloat(r.FormValue("rate"))
	if err != nil {
		LoggerFromContext(r.Context()).Warn("create rate: invalid rate", "value", r.FormValue("rate"))
		http.Error(w, "rate must be a positive number", http.StatusBadRequest)
		return
	}

	if err := store.CreateRate(category, startDate, endDate, rateValue); err != nil {
		LoggerFromContext(r.Context()).Error("create rate", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	redirect(w, r, "/rates", "Rate added")
}

func (h *RateHandlers) deleteRate(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid rate id", http.StatusBadRequest)
		return
	}
	if err := store.DeleteRate(id); err != nil {
		LoggerFromContext(r.Context()).Error("delete rate", "rate_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !isHTMX(r) {
		redirect(w, r, "/rates", "Rate deleted")
		return
	}

	activeOnly := activeRatesOnly(r)
	rates, err := listRates(r, activeOnly)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list rates after delete", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Partial(w, http.StatusOK, "rates_table", RatesPageData{Rates: rates, Notice: fmt.Sprintf("Rate %d deleted", id), ActiveOnly: activeOnly}, "rates_table.html")
}

func (h *RateHandlers) editRateForm(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid rate id", http.StatusBadRequest)
		return
	}
	rate, err := store.GetRate(id)
	if err != nil {
		LoggerFromContext(r.Context()).Error("get rate for edit", "rate_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Partial(w, http.StatusOK, "rate_edit_row", rate, "rate_edit_row.html")
}

func (h *RateHandlers) updateRate(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid rate id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		LoggerFromContext(r.Context()).Warn("update rate: missing category")
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	startDate, err := validateDate(r.FormValue("start_date"))
	if err != nil {
		LoggerFromContext(r.Context()).Warn("update rate: invalid start date", "value", r.FormValue("start_date"))
		http.Error(w, "start date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	endDate := strings.TrimSpace(r.FormValue("end_date"))
	if endDate != "" {
		var err error
		endDate, err = validateDate(endDate)
		if err != nil {
			LoggerFromContext(r.Context()).Warn("update rate: invalid end date", "value", r.FormValue("end_date"))
			http.Error(w, "end date must use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}
	if endDate != "" && endDate < startDate {
		LoggerFromContext(r.Context()).Warn("update rate: end before start", "start_date", startDate, "end_date", endDate)
		http.Error(w, "end date must be on or after start date", http.StatusBadRequest)
		return
	}
	rateValue, err := parsePositiveFloat(r.FormValue("rate"))
	if err != nil {
		LoggerFromContext(r.Context()).Warn("update rate: invalid rate", "value", r.FormValue("rate"))
		http.Error(w, "rate must be a positive number", http.StatusBadRequest)
		return
	}

	if err := store.UpdateRate(id, category, startDate, endDate, rateValue); err != nil {
		LoggerFromContext(r.Context()).Error("update rate", "rate_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	activeOnly := activeRatesOnly(r)
	rates, err := listRates(r, activeOnly)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list rates after update", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Partial(w, http.StatusOK, "rates_table", RatesPageData{Rates: rates, Notice: fmt.Sprintf("Rate %d updated", id), ActiveOnly: activeOnly}, "rates_table.html")
}

func (h *RateHandlers) ratesTable(w http.ResponseWriter, r *http.Request) {
	activeOnly := activeRatesOnly(r)
	rates, err := listRates(r, activeOnly)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list rates table", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Partial(w, http.StatusOK, "rates_table", RatesPageData{Rates: rates, ActiveOnly: activeOnly}, "rates_table.html")
}

func activeRatesOnly(r *http.Request) bool {
	value := strings.ToLower(strings.TrimSpace(r.FormValue("active_only")))
	return value == "1" || value == "on" || value == "true" || value == "yes"
}

func listRates(r *http.Request, activeOnly bool) ([]db.Rate, error) {
	store := StoreFromContext(r.Context())
	if !activeOnly {
		return store.ListRates()
	}
	return store.ListActiveRates(time.Now().Format("2006-01-02"))
}
