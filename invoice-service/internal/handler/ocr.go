package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/ocrimport"
)

const ocrMaxRequestBytes = 64 << 20

type OCRHandlers struct {
	renderer *Renderer
	ocrJobs  *OCRJobRunner
	cfg      config.Config
}

func NewOCRHandlers(renderer *Renderer, ocrJobs *OCRJobRunner, cfg config.Config) *OCRHandlers {
	return &OCRHandlers{renderer: renderer, ocrJobs: ocrJobs, cfg: cfg}
}

func (h *OCRHandlers) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ocr/import", h.ocrUploadPage)
	mux.HandleFunc("POST /ocr/import", h.ocrStartSession)
	mux.HandleFunc("GET /ocr/import/{id}", h.ocrSessionStatus)
	mux.HandleFunc("POST /ocr/import/{id}/confirm", h.ocrConfirmDrafts)
	mux.HandleFunc("POST /ocr/import/{id}/drafts/{draftID}/delete", h.ocrDeleteDraft)
	mux.HandleFunc("POST /ocr/import/{id}/delete", h.ocrDeleteSession)
}

func (h *OCRHandlers) ocrUploadPage(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.OCREnabled {
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

	h.renderer.Page(w, r, http.StatusOK, "ocr_upload.html", data)
}

func (h *OCRHandlers) ocrStartSession(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.OCREnabled {
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

	importer := h.ocrImporter(store)
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
	if h.cfg.OCRServiceURL != "" {
		user := UserFromContext(r.Context())
		if user != nil {
			h.ocrJobs.Start(r.Context(), sessionID, user.ID)
		}
	}
	redirect(w, r, "/ocr/import/"+strconv.FormatInt(sessionID, 10), "Import session created")
}

func (h *OCRHandlers) ocrSessionStatus(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.OCREnabled {
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
	categories = mergeDraftCategories(categories, drafts)

	data := db.DraftReviewData{
		Session:    session,
		Images:     images,
		Drafts:     drafts,
		Categories: categories,
		Notice:     noticeFromRequest(r),
	}

	h.renderer.Page(w, r, http.StatusOK, "ocr_review.html", data)
}

func (h *OCRHandlers) ocrConfirmDrafts(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.OCREnabled {
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
		redirect(w, r, fmt.Sprintf("/ocr/import/%d", sessionID), "Select at least one entry to confirm")
		return
	}

	importer := h.ocrImporter(store)
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
		redirect(w, r, fmt.Sprintf("/ocr/import/%d", sessionID), err.Error())
		return
	}

	notice := fmt.Sprintf("Confirmed %d entries from import", len(result.Confirmed))
	if len(result.Skipped) > 0 {
		notice += fmt.Sprintf(", %d skipped (missing data)", len(result.Skipped))
	}
	redirect(w, r, "/entries", notice)
}

func mergeDraftCategories(categories []string, drafts []db.OCRDraftEntry) []string {
	seen := make(map[string]bool, len(categories)+len(drafts))
	merged := make([]string, 0, len(categories)+len(drafts))
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category == "" || seen[category] {
			continue
		}
		seen[category] = true
		merged = append(merged, category)
	}
	for _, draft := range drafts {
		for _, category := range []string{draft.CategoryNormalized, draft.CategoryRaw} {
			category = strings.TrimSpace(category)
			if category == "" || seen[category] {
				continue
			}
			seen[category] = true
			merged = append(merged, category)
		}
	}
	return merged
}

func (h *OCRHandlers) ocrDeleteDraft(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())

	sessionID, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	draftID, err := parseInt64Path(r, "draftID")
	if err != nil {
		http.Error(w, "invalid draft id", http.StatusBadRequest)
		return
	}

	importer := h.ocrImporter(store)
	if err := importer.DeleteDraft(r.Context(), sessionID, draftID); err != nil {
		LoggerFromContext(r.Context()).Warn("delete ocr draft", "session_id", sessionID, "draft_id", draftID, "error", err)
		if isHTMX(r) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		redirect(w, r, fmt.Sprintf("/ocr/import/%d", sessionID), err.Error())
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	redirect(w, r, fmt.Sprintf("/ocr/import/%d", sessionID), "Draft entry deleted")
}

func (h *OCRHandlers) ocrDeleteSession(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.OCREnabled {
		http.Error(w, "OCR import is not enabled", http.StatusNotFound)
		return
	}

	store := StoreFromContext(r.Context())

	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	importer := h.ocrImporter(store)
	if err := importer.DeleteSession(r.Context(), id); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirect(w, r, "/ocr/import", "Import session deleted")
}

func (h *OCRHandlers) ocrImporter(store ocrimport.Store) *ocrimport.Importer {
	return h.ocrJobs.importer(store)
}
