package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"invoice-app/internal/db"
	"invoice-app/internal/ocr"
)

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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "form too large", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "at least one image is required", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(a.cfg.OCRUploadDir, 0o755); err != nil {
		http.Error(w, "create upload directory", http.StatusInternalServerError)
		return
	}

	sessionID, err := store.CreateOCRSession()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var uploadedFiles []string
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			store.DeleteOCRSession(sessionID)
			for _, f := range uploadedFiles {
				os.Remove(f)
				os.Remove(f + ".processed.jpg")
			}
		}
	}()

	for _, header := range files {
		ext := filepath.Ext(header.Filename)
		savedName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
		savedPath := filepath.Join(a.cfg.OCRUploadDir, savedName)

		src, err := header.Open()
		if err != nil {
			http.Error(w, fmt.Sprintf("open uploaded file: %v", err), http.StatusInternalServerError)
			return
		}

		dst, err := os.Create(savedPath)
		if err != nil {
			src.Close()
			http.Error(w, fmt.Sprintf("save uploaded file: %v", err), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			http.Error(w, fmt.Sprintf("copy uploaded file: %v", err), http.StatusInternalServerError)
			return
		}
		src.Close()
		dst.Close()

		if err := ocr.ValidateImageFile(savedPath); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		processedPath := savedPath
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			processedPath = savedPath + ".processed.jpg"
			if err := ocr.PreprocessImage(savedPath, processedPath); err != nil {
				processedPath = savedPath
			}
		}

		info, _ := os.Stat(processedPath)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}

		if err := store.AddSessionImage(sessionID, header.Filename, processedPath, size); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		uploadedFiles = append(uploadedFiles, savedPath)
	}

	if err := store.UpdateOCRSessionState(sessionID, "uploaded", ""); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if a.cfg.OCRServiceURL != "" {
		user := UserFromContext(r.Context())
		if user != nil {
			a.ocrWg.Add(1)
			go a.processOCRSession(sessionID, user.ID)
		}
	}

	cleanupNeeded = false
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

	for _, id := range confirmedIDs {
		if e, ok := edits[id]; ok {
			if err := store.UpdateDraftEntry(id, e.date, e.category, e.hours, e.notes); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}
	}

	confirmed, skipped, err := store.ConfirmDraftEntries(sessionID, confirmedIDs)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	notice := fmt.Sprintf("Confirmed %d entries from import", len(confirmed))
	if len(skipped) > 0 {
		notice += fmt.Sprintf(", %d skipped (missing data)", len(skipped))
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

	images, _ := store.GetSessionImages(id)
	for _, img := range images {
		if img.FilePath != "" {
			os.Remove(img.FilePath)
			os.Remove(img.FilePath + ".processed.jpg")
		}
	}

	if err := store.DeleteOCRSession(id); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.redirect(w, r, "/ocr/import", "Import session deleted")
}

func (a *App) processOCRSession(sessionID int64, userID int64) {
	defer a.ocrWg.Done()

	store, err := a.multiStore.ForUser(userID)
	if err != nil {
		return
	}

	if err := store.UpdateOCRSessionState(sessionID, "processing", ""); err != nil {
		return
	}

	images, err := store.GetSessionImages(sessionID)
	if err != nil {
		store.UpdateOCRSessionState(sessionID, "failed", fmt.Sprintf("get images: %v", err))
		return
	}

	var imagePaths []string
	for _, img := range images {
		imagePaths = append(imagePaths, img.FilePath)
	}

	categories, err := store.ListCategories()
	if err != nil {
		store.UpdateOCRSessionState(sessionID, "failed", fmt.Sprintf("list categories: %v", err))
		return
	}

	rates, err := store.ListRates()
	if err != nil {
		store.UpdateOCRSessionState(sessionID, "failed", fmt.Sprintf("list rates: %v", err))
		return
	}

	var rateHints []ocr.RateHint
	for _, r := range rates {
		rateHints = append(rateHints, ocr.RateHint{
			Category:  r.Category,
			StartDate: r.StartDate,
			EndDate:   r.EndDate,
			Rate:      r.Rate,
		})
	}

	var result *ocr.OCRResponse
	if a.cfg.OCRServiceURL != "" {
		client := ocr.NewClient(a.cfg.OCRServiceURL)
		result, err = client.Extract(imagePaths, ocr.ContextHint{
			Categories: categories,
			SendRates:  true,
		}, rateHints, sessionID)
	} else {
		stub := ocr.NewStubClient()
		result, err = stub.Extract(imagePaths, ocr.ContextHint{
			Categories: categories,
			SendRates:  true,
		}, rateHints, sessionID)
	}

	if err != nil {
		store.UpdateOCRSessionState(sessionID, "failed", fmt.Sprintf("OCR processing failed: %v", err))
		return
	}

	knownCategories := make(map[string]bool)
	for _, cat := range categories {
		knownCategories[cat] = true
	}

	var drafts []db.OCRDraftEntry
	for _, e := range result.Entries {
		if e.Category.Raw != "" && e.Category.Normalized == "" {
			normalized := ocr.NormalizeOCREntry(e.Category.Raw, "category", knownCategories)
			e.Category = normalized
		}

		hoursVal := 0.0
		if h, err := strconv.ParseFloat(e.Hours.Normalized, 64); err == nil && h > 0 {
			hoursVal = h
		}

		drafts = append(drafts, db.OCRDraftEntry{
			DateRaw:            e.Date.Raw,
			DateNormalized:     e.Date.Normalized,
			CategoryRaw:        e.Category.Raw,
			CategoryNormalized: e.Category.Normalized,
			HoursRaw:           e.Hours.Raw,
			HoursNormalized:    hoursVal,
			NotesRaw:           e.Notes.Raw,
			NotesNormalized:    e.Notes.Normalized,
			Confidence:         e.Category.Confidence,
			NeedsReview:        e.Category.NeedsReview,
		})
	}

	if err := store.SaveDraftEntries(sessionID, drafts); err != nil {
		store.UpdateOCRSessionState(sessionID, "failed", fmt.Sprintf("save drafts: %v", err))
		return
	}

	store.UpdateOCRSessionState(sessionID, "review_ready", "")
}
