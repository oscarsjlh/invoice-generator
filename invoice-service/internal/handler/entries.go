package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type EntryHandlers struct {
	renderer *Renderer
}

func NewEntryHandlers(renderer *Renderer) *EntryHandlers {
	return &EntryHandlers{renderer: renderer}
}

func (h *EntryHandlers) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /entries", h.entriesPage)
	mux.HandleFunc("GET /entries/table", h.entriesTable)
	mux.HandleFunc("POST /entries", h.createEntry)
	mux.HandleFunc("GET /entries/{id}/edit", h.editEntryForm)
	mux.HandleFunc("POST /entries/{id}", h.updateEntry)
	mux.HandleFunc("POST /entries/{id}/delete", h.deleteEntry)
}

func (h *EntryHandlers) entriesPage(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	year, month, rate := entryFiltersFromRequest(r)
	entries, err := store.ListEntriesFiltered(year, month, rate)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entries", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	categories, err := store.ListCategories()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list categories", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	years, months, err := entryFilterOptions(store, year)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entry filters", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Page(w, r, http.StatusOK, "entries.html", EntriesPageData{
		Entries:       entries,
		Categories:    categories,
		Years:         years,
		Months:        months,
		SelectedYear:  year,
		SelectedMonth: month,
		SelectedRate:  rate,
		FilterQuery:   entryFilterQuery(year, month, rate),
		Notice:        noticeFromRequest(r),
	}, "entries_table.html")
}

func (h *EntryHandlers) createEntry(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	date, err := validateDate(r.FormValue("date"))
	if err != nil {
		LoggerFromContext(r.Context()).Warn("create entry: invalid date", "value", r.FormValue("date"))
		http.Error(w, "date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		LoggerFromContext(r.Context()).Warn("create entry: missing category")
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	hours, err := parsePositiveFloat(r.FormValue("hours"))
	if err != nil {
		LoggerFromContext(r.Context()).Warn("create entry: invalid hours", "value", r.FormValue("hours"))
		http.Error(w, "hours must be a positive number", http.StatusBadRequest)
		return
	}
	if ok, err := entryCategoryExists(store, category); err != nil {
		LoggerFromContext(r.Context()).Error("check entry category", "category", category, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	} else if !ok {
		LoggerFromContext(r.Context()).Warn("create entry: category does not exist", "category", category)
		redirect(w, r, entriesPathWithNotice(r, entryCategoryMissingNotice(category)), "")
		return
	}

	if err := store.CreateEntry(date, category, hours, strings.TrimSpace(r.FormValue("notes"))); err != nil {
		LoggerFromContext(r.Context()).Error("create entry", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	LoggerFromContext(r.Context()).Info("entry created", "date", date, "category", category)
	redirect(w, r, entriesPathWithNotice(r, ""), "Entry added")
}

func (h *EntryHandlers) editEntryForm(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid entry id", http.StatusBadRequest)
		return
	}
	entry, err := store.GetEntry(id)
	if err != nil {
		LoggerFromContext(r.Context()).Error("get entry for edit", "entry_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	categories, err := store.ListCategories()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list categories for edit", "entry_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	year, month, rate := entryFiltersFromRequest(r)
	h.renderer.Partial(w, http.StatusOK, "entry_edit_row", EntryEditRowData{
		Entry:       entry,
		Categories:  categories,
		FilterQuery: entryFilterQuery(year, month, rate),
	}, "entry_edit_row.html")
}

func (h *EntryHandlers) updateEntry(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid entry id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	date, err := validateDate(r.FormValue("date"))
	if err != nil {
		http.Error(w, "date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	hours, err := parsePositiveFloat(r.FormValue("hours"))
	if err != nil {
		http.Error(w, "hours must be a positive number", http.StatusBadRequest)
		return
	}
	if ok, err := entryCategoryExists(store, category); err != nil {
		LoggerFromContext(r.Context()).Error("check entry category", "entry_id", id, "category", category, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	} else if !ok {
		LoggerFromContext(r.Context()).Warn("update entry: category does not exist", "entry_id", id, "category", category)
		h.renderEntriesTable(w, r, http.StatusOK, entryCategoryMissingNotice(category))
		return
	}

	if err := store.UpdateEntry(id, date, category, hours, strings.TrimSpace(r.FormValue("notes"))); err != nil {
		LoggerFromContext(r.Context()).Error("update entry", "entry_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.renderEntriesTable(w, r, http.StatusOK, fmt.Sprintf("Entry %d updated", id))
}

func (h *EntryHandlers) entriesTable(w http.ResponseWriter, r *http.Request) {
	h.renderEntriesTable(w, r, http.StatusOK, "")
}

func (h *EntryHandlers) deleteEntry(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid entry id", http.StatusBadRequest)
		return
	}
	if err := store.DeleteEntry(id); err != nil {
		LoggerFromContext(r.Context()).Error("delete entry", "entry_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !isHTMX(r) {
		redirect(w, r, "/entries", "Entry deleted")
		return
	}

	h.renderEntriesTable(w, r, http.StatusOK, fmt.Sprintf("Entry %d deleted", id))
}

func (h *EntryHandlers) renderEntriesTable(w http.ResponseWriter, r *http.Request, status int, notice string) {
	store := StoreFromContext(r.Context())
	year, month, rate := entryFiltersFromRequest(r)
	entries, err := store.ListEntriesFiltered(year, month, rate)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entries table", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	years, months, err := entryFilterOptions(store, year)
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entry filters", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	categories, err := store.ListCategories()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list categories for entry filters", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Partial(w, status, "entries_table", EntriesPageData{
		Entries:       entries,
		Categories:    categories,
		Years:         years,
		Months:        months,
		SelectedYear:  year,
		SelectedMonth: month,
		SelectedRate:  rate,
		FilterQuery:   entryFilterQuery(year, month, rate),
		Notice:        notice,
	}, "entries_table.html")
}

func entryFilterOptions(store interface {
	ListAvailableYears() ([]string, error)
	ListAvailableMonths(string) ([]string, error)
}, year string) ([]string, []string, error) {
	years, err := store.ListAvailableYears()
	if err != nil {
		return nil, nil, err
	}
	months, err := store.ListAvailableMonths(year)
	if err != nil {
		return nil, nil, err
	}
	return years, months, nil
}

func entryFiltersFromRequest(r *http.Request) (string, string, string) {
	if err := r.ParseForm(); err != nil {
		return "", "", ""
	}
	return normalizeEntryFilters(r.FormValue("year"), r.FormValue("month"), r.FormValue("rate"))
}

func normalizeEntryFilters(year, month, rate string) (string, string, string) {
	year = strings.TrimSpace(year)
	month = strings.TrimSpace(month)
	rate = strings.TrimSpace(rate)
	if len(month) == len("2006-01") {
		if parsed, err := validateMonth(month); err == nil {
			if year == "" {
				year = parsed[:4]
			}
			month = parsed[5:]
		}
	}
	if len(year) != 4 {
		year = ""
	} else if _, err := strconv.Atoi(year); err != nil {
		year = ""
	}
	if len(month) != 2 {
		month = ""
	} else if parsed, err := strconv.Atoi(month); err != nil || parsed < 1 || parsed > 12 {
		month = ""
	}
	return year, month, rate
}

func entryFilterQuery(year, month, rate string) string {
	values := url.Values{}
	if year != "" {
		values.Set("year", year)
	}
	if month != "" {
		values.Set("month", month)
	}
	if rate != "" {
		values.Set("rate", rate)
	}
	if encoded := values.Encode(); encoded != "" {
		return "?" + encoded
	}
	return ""
}

func entriesPathWithNotice(r *http.Request, notice string) string {
	year, month, rate := entryFiltersFromRequest(r)
	values := url.Values{}
	if year != "" {
		values.Set("year", year)
	}
	if month != "" {
		values.Set("month", month)
	}
	if rate != "" {
		values.Set("rate", rate)
	}
	if notice != "" {
		values.Set("notice", notice)
	}
	if encoded := values.Encode(); encoded != "" {
		return "/entries?" + encoded
	}
	return "/entries"
}

func entryCategoryExists(store interface{ ListCategories() ([]string, error) }, category string) (bool, error) {
	categories, err := store.ListCategories()
	if err != nil {
		return false, err
	}
	for _, existing := range categories {
		if existing == category {
			return true, nil
		}
	}
	return false, nil
}

func entryCategoryMissingNotice(category string) string {
	return fmt.Sprintf("Category %q does not exist. Add a rate for this category before saving an entry.", category)
}
