package handler

import (
	"fmt"
	"net/http"
	"strings"
)

func (a *App) entriesPage(w http.ResponseWriter, r *http.Request) {
	entries, err := a.store.ListEntries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	categories, err := a.store.ListCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPage(w, http.StatusOK, "entries.html", EntriesPageData{
		Entries:    entries,
		Categories: categories,
		Notice:     noticeFromRequest(r),
	}, "entries_table.html")
}

func (a *App) createEntry(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	if err := a.store.CreateEntry(date, category, hours, strings.TrimSpace(r.FormValue("notes"))); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.redirect(w, r, "/entries", "Entry added")
}

func (a *App) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid entry id", http.StatusBadRequest)
		return
	}
	if err := a.store.DeleteEntry(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isHTMX(r) {
		a.redirect(w, r, "/entries", "Entry deleted")
		return
	}

	entries, err := a.store.ListEntries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "entries_table", EntriesPageData{Entries: entries, Notice: fmt.Sprintf("Entry %d deleted", id)}, "entries_table.html")
}
