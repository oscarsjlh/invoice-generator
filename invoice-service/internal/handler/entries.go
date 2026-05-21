package handler

import (
	"fmt"
	"net/http"
	"strings"
)

func (a *App) entriesPage(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	entries, err := store.ListEntries()
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
	a.renderPage(w, r, http.StatusOK, "entries.html", EntriesPageData{
		Entries:    entries,
		Categories: categories,
		Notice:     noticeFromRequest(r),
	}, "entries_table.html")
}

func (a *App) createEntry(w http.ResponseWriter, r *http.Request) {
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

	if err := store.CreateEntry(date, category, hours, strings.TrimSpace(r.FormValue("notes"))); err != nil {
		LoggerFromContext(r.Context()).Error("create entry", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	LoggerFromContext(r.Context()).Info("entry created", "date", date, "category", category)
	a.redirect(w, r, "/entries", "Entry added")
}

func (a *App) editEntryForm(w http.ResponseWriter, r *http.Request) {
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
	a.renderPartial(w, http.StatusOK, "entry_edit_row", entry, "entry_edit_row.html")
}

func (a *App) updateEntry(w http.ResponseWriter, r *http.Request) {
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

	if err := store.UpdateEntry(id, date, category, hours, strings.TrimSpace(r.FormValue("notes"))); err != nil {
		LoggerFromContext(r.Context()).Error("update entry", "entry_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	entries, err := store.ListEntries()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entries after update", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "entries_table", EntriesPageData{Entries: entries, Notice: fmt.Sprintf("Entry %d updated", id)}, "entries_table.html")
}

func (a *App) entriesTable(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	entries, err := store.ListEntries()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entries table", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "entries_table", EntriesPageData{Entries: entries}, "entries_table.html")
}

func (a *App) deleteEntry(w http.ResponseWriter, r *http.Request) {
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
		a.redirect(w, r, "/entries", "Entry deleted")
		return
	}

	entries, err := store.ListEntries()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list entries after delete", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	a.renderPartial(w, http.StatusOK, "entries_table", EntriesPageData{Entries: entries, Notice: fmt.Sprintf("Entry %d deleted", id)}, "entries_table.html")
}
