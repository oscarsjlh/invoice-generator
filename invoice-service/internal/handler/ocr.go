package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"invoice-app/internal/db"
	"invoice-app/internal/ocrimport"
)

const ocrMaxRequestBytes = 64 << 20

func (a *App) ocrUploadPage(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())
	sessions, err := store.ListOCRSessions()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Sessions []db.OCRSession
		Notice   string
		User     *db.User
	}{
		Sessions: sessions,
		Notice:   noticeFromRequest(r),
	}

	a.renderPage(w, r, http.StatusOK, "ocr_upload.html", data)
}

func (a *App) ocrStartSession(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, ocrMaxRequestBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "form too large", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "at least one image is required", http.StatusBadRequest)
		return
	}

	importer := a.ocrImporter(store)
	sessionID, err := importer.StartSession(r.Context(), ocrimport.StartSessionInput{Files: files})
	if errors.Is(err, ocrimport.ErrTooManyFiles) || errors.Is(err, ocrimport.ErrNoImages) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		LoggerFromContext(r.Context()).Error("start ocr session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if a.cfg.OCRServiceURL != "" {
		user := UserFromContext(r.Context())
		if user != nil {
			a.ocrJobs.Start(sessionID, user.ID)
		}
	}
	a.redirect(w, r, "/ocr/import/"+strconv.FormatInt(sessionID, 10), "Import session created")
}

func (a *App) ocrSessionStatus(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())

	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	session, err := store.GetOCRSession(id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	images, err := store.GetSessionImages(id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	drafts, err := store.GetDraftEntries(id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	categories, err := store.ListCategories()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := db.DraftReviewData{
		Session:    session,
		Images:     images,
		Drafts:     drafts,
		Categories: categories,
		Notice:     noticeFromRequest(r),
	}

	a.renderPage(w, r, http.StatusOK, "ocr_review.html", data)
}

func (a *App) ocrConfirmDrafts(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())

	sessionID, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	type edit struct {
		date     string
		category string
		hours    string
		notes    string
	}
	edits := make(map[int64]edit)
	var confirmedIDs []int64

	for key, values := range r.PostForm {
		if len(values) == 0 {
			continue
		}
		val := values[0]
		var draftID int64
		if _, err := fmt.Sscanf(key, "draft_%d", &draftID); err == nil {
			if val == "on" {
				confirmedIDs = append(confirmedIDs, draftID)
			}
		} else if _, err := fmt.Sscanf(key, "date_%d", &draftID); err == nil {
			e := edits[draftID]
			e.date = val
			edits[draftID] = e
		} else if _, err := fmt.Sscanf(key, "cat_%d", &draftID); err == nil {
			e := edits[draftID]
			e.category = val
			edits[draftID] = e
		} else if _, err := fmt.Sscanf(key, "hrs_%d", &draftID); err == nil {
			e := edits[draftID]
			e.hours = val
			edits[draftID] = e
		} else if _, err := fmt.Sscanf(key, "notes_%d", &draftID); err == nil {
			e := edits[draftID]
			e.notes = val
			edits[draftID] = e
		}
	}

	if len(confirmedIDs) == 0 {
		a.redirect(w, r, fmt.Sprintf("/ocr/import/%d", sessionID), "Select at least one entry to confirm")
		return
	}

	importer := a.ocrImporter(store)
	ocrEdits := make(map[int64]ocrimport.DraftEdit, len(edits))
	for id, e := range edits {
		ocrEdits[id] = ocrimport.DraftEdit{
			Date:     e.date,
			Category: e.category,
			Hours:    e.hours,
			Notes:    e.notes,
		}
	}
	result, err := importer.ConfirmDrafts(r.Context(), ocrimport.ConfirmDraftsInput{
		SessionID: sessionID,
		IDs:       confirmedIDs,
		Edits:     ocrEdits,
	})
	if err != nil {
		LoggerFromContext(r.Context()).Warn("confirm ocr drafts", "session_id", sessionID, "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	notice := fmt.Sprintf("Confirmed %d entries from import", len(result.Confirmed))
	if len(result.Skipped) > 0 {
		notice += fmt.Sprintf(", %d skipped (missing data)", len(result.Skipped))
	}
	a.redirect(w, r, "/entries", notice)
}

func (a *App) ocrDeleteSession(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())

	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	importer := a.ocrImporter(store)
	if err := importer.DeleteSession(r.Context(), id); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.redirect(w, r, "/ocr/import", "Import session deleted")
}

func (a *App) ocrImporter(store ocrimport.Store) *ocrimport.Importer {
	return a.ocrJobs.importer(store)
}
