package ocrimport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"invoice-app/internal/db"
	"invoice-app/internal/ocr"
)

const (
	DefaultMaxFiles = 10
)

var (
	ErrNoImages     = errors.New("at least one image is required")
	ErrTooManyFiles = errors.New("too many images")
)

type Store interface {
	CreateOCRSession() (int64, error)
	GetOCRSession(id int64) (db.OCRSession, error)
	UpdateOCRSessionState(id int64, state string, errorMsg string) error
	ListOCRSessions() ([]db.OCRSession, error)
	AddSessionImage(sessionID int64, fileName, filePath string, fileSize int64) error
	GetSessionImages(sessionID int64) ([]db.OCRSessionImage, error)
	SaveDraftEntries(sessionID int64, drafts []db.OCRDraftEntry) error
	GetDraftEntries(sessionID int64) ([]db.OCRDraftEntry, error)
	GetDraftEntry(id int64) (db.OCRDraftEntry, error)
	UpdateDraftEntry(id int64, date, category, hours, notes string) error
	ConfirmDraftEntries(sessionID int64, ids []int64) (confirmed []int64, skipped []int64, err error)
	DeleteOCRSession(id int64) error
	ListCategories() ([]string, error)
	ListRates() ([]db.Rate, error)
}

type Importer struct {
	Store     Store
	Extractor ocr.Extractor
	UploadDir string
	MaxFiles  int
}

type StartSessionInput struct {
	Files []*multipart.FileHeader
}

type ConfirmDraftsInput struct {
	SessionID int64
	IDs       []int64
	Edits     map[int64]DraftEdit
}

type DraftEdit struct {
	Date     string
	Category string
	Hours    string
	Notes    string
}

type ConfirmResult struct {
	Confirmed []int64
	Skipped   []int64
}

func (i *Importer) StartSession(ctx context.Context, input StartSessionInput) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(input.Files) == 0 {
		return 0, ErrNoImages
	}
	maxFiles := i.MaxFiles
	if maxFiles <= 0 {
		maxFiles = DefaultMaxFiles
	}
	if len(input.Files) > maxFiles {
		return 0, ErrTooManyFiles
	}

	if err := os.MkdirAll(i.UploadDir, 0o755); err != nil {
		return 0, fmt.Errorf("create upload directory: %w", err)
	}

	sessionID, err := i.Store.CreateOCRSession()
	if err != nil {
		return 0, err
	}

	var uploadedFiles []string
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			_ = i.Store.DeleteOCRSession(sessionID)
			for _, f := range uploadedFiles {
				_ = os.Remove(f)
				_ = os.Remove(f + ".processed.jpg")
			}
		}
	}()

	for _, header := range input.Files {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		savedPath, processedPath, err := i.saveAndPreprocess(header)
		if err != nil {
			return 0, err
		}
		uploadedFiles = append(uploadedFiles, savedPath)

		info, _ := os.Stat(processedPath)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}

		if err := i.Store.AddSessionImage(sessionID, header.Filename, processedPath, size); err != nil {
			return 0, err
		}
	}

	if err := i.Store.UpdateOCRSessionState(sessionID, "uploaded", ""); err != nil {
		return 0, err
	}

	cleanupNeeded = false
	return sessionID, nil
}

func (i *Importer) saveAndPreprocess(header *multipart.FileHeader) (savedPath string, processedPath string, err error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	savedName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(header.Filename))
	savedPath = filepath.Join(i.UploadDir, savedName)

	src, err := header.Open()
	if err != nil {
		return "", "", fmt.Errorf("open uploaded file: %w", err)
	}
	defer func() {
		_ = src.Close()
	}()

	dst, err := os.Create(savedPath)
	if err != nil {
		return "", "", fmt.Errorf("save uploaded file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return "", "", fmt.Errorf("copy uploaded file: %w", err)
	}
	if err := dst.Close(); err != nil {
		return "", "", fmt.Errorf("finalize uploaded file: %w", err)
	}

	if err := ocr.ValidateImageFile(savedPath); err != nil {
		return "", "", err
	}

	processedPath = savedPath
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		processedPath = savedPath + ".processed.jpg"
		if err := ocr.PreprocessImage(savedPath, processedPath); err != nil {
			processedPath = savedPath
		}
	}
	return savedPath, processedPath, nil
}

func (i *Importer) ProcessSession(ctx context.Context, sessionID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := i.Store.UpdateOCRSessionState(sessionID, "processing", ""); err != nil {
		return err
	}

	images, err := i.Store.GetSessionImages(sessionID)
	if err != nil {
		return i.fail(sessionID, "get images", err)
	}
	imagePaths := make([]string, 0, len(images))
	for _, img := range images {
		imagePaths = append(imagePaths, img.FilePath)
	}

	categories, err := i.Store.ListCategories()
	if err != nil {
		return i.fail(sessionID, "list categories", err)
	}
	rates, err := i.Store.ListRates()
	if err != nil {
		return i.fail(sessionID, "list rates", err)
	}

	result, err := i.extract(imagePaths, categories, rates, sessionID)
	if err != nil {
		return i.fail(sessionID, "OCR processing failed", err)
	}

	drafts := buildDraftEntries(result, categories)
	if err := i.Store.SaveDraftEntries(sessionID, drafts); err != nil {
		return i.fail(sessionID, "save drafts", err)
	}

	return i.Store.UpdateOCRSessionState(sessionID, "review_ready", "")
}

func (i *Importer) extract(imagePaths []string, categories []string, rates []db.Rate, sessionID int64) (*ocr.OCRResponse, error) {
	rateHints := make([]ocr.RateHint, 0, len(rates))
	for _, r := range rates {
		rateHints = append(rateHints, ocr.RateHint{
			Category:  r.Category,
			StartDate: r.StartDate,
			EndDate:   r.EndDate,
			Rate:      r.Rate,
		})
	}

	extractor := i.Extractor
	if extractor == nil {
		extractor = ocr.NewStubClient()
	}
	return extractor.Extract(imagePaths, ocr.ContextHint{
		Categories: categories,
		SendRates:  true,
	}, rateHints, sessionID)
}

func buildDraftEntries(result *ocr.OCRResponse, categories []string) []db.OCRDraftEntry {
	knownCategories := make(map[string]bool, len(categories))
	for _, cat := range categories {
		knownCategories[cat] = true
	}

	drafts := make([]db.OCRDraftEntry, 0, len(result.Entries))
	for _, e := range result.Entries {
		if e.Category.Raw != "" && e.Category.Normalized == "" {
			e.Category = ocr.NormalizeOCREntry(e.Category.Raw, "category", knownCategories)
		}

		hoursVal := 0.0
		if h, err := strconv.ParseFloat(e.Hours.Normalized, 64); err == nil && h > 0 {
			hoursVal = h
		}

		drafts = append(drafts, db.OCRDraftEntry{
			DateRaw:            e.Date.Raw,
			DateNormalized:     e.Date.Normalized,
			CategoryRaw:        e.Category.Raw,
			CategoryNormalized: strings.TrimSpace(e.Category.Normalized),
			HoursRaw:           e.Hours.Raw,
			HoursNormalized:    hoursVal,
			NotesRaw:           e.Notes.Raw,
			NotesNormalized:    strings.TrimSpace(e.Notes.Normalized),
			Confidence:         e.Category.Confidence,
			NeedsReview:        e.Category.NeedsReview,
		})
	}
	return drafts
}

func (i *Importer) ConfirmDrafts(ctx context.Context, input ConfirmDraftsInput) (ConfirmResult, error) {
	if err := ctx.Err(); err != nil {
		return ConfirmResult{}, err
	}
	for _, id := range input.IDs {
		edit, hasEdit := input.Edits[id]
		if hasEdit {
			date, category, hours, notes, err := validateDraftValues(edit.Date, edit.Category, edit.Hours, edit.Notes)
			if err != nil {
				return ConfirmResult{}, err
			}
			if err := i.Store.UpdateDraftEntry(id, date, category, hours, notes); err != nil {
				return ConfirmResult{}, err
			}
		}

		draft, err := i.Store.GetDraftEntry(id)
		if err != nil {
			return ConfirmResult{}, err
		}
		if draft.SessionID != input.SessionID {
			return ConfirmResult{}, fmt.Errorf("draft %d does not belong to session %d", id, input.SessionID)
		}
		if err := validateDraftEntry(draft); err != nil {
			return ConfirmResult{}, err
		}
	}

	confirmed, skipped, err := i.Store.ConfirmDraftEntries(input.SessionID, input.IDs)
	if err != nil {
		return ConfirmResult{}, err
	}
	return ConfirmResult{Confirmed: confirmed, Skipped: skipped}, nil
}

func (i *Importer) DeleteSession(ctx context.Context, sessionID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	images, _ := i.Store.GetSessionImages(sessionID)
	for _, img := range images {
		if img.FilePath != "" {
			_ = os.Remove(img.FilePath)
			_ = os.Remove(img.FilePath + ".processed.jpg")
		}
	}
	return i.Store.DeleteOCRSession(sessionID)
}

func (i *Importer) fail(sessionID int64, action string, err error) error {
	msg := fmt.Sprintf("%s: %v", action, err)
	_ = i.Store.UpdateOCRSessionState(sessionID, "failed", msg)
	return fmt.Errorf("%s: %w", action, err)
}

func validateDraftValues(date, category, hours, notes string) (string, string, string, string, error) {
	date = strings.TrimSpace(date)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", "", "", "", fmt.Errorf("date must use YYYY-MM-DD")
	}
	category = strings.TrimSpace(category)
	if category == "" {
		return "", "", "", "", fmt.Errorf("category is required")
	}
	hours = strings.TrimSpace(hours)
	hoursVal, err := strconv.ParseFloat(hours, 64)
	if err != nil || hoursVal <= 0 {
		return "", "", "", "", fmt.Errorf("hours must be a positive number")
	}
	return date, category, hours, strings.TrimSpace(notes), nil
}

func validateDraftEntry(draft db.OCRDraftEntry) error {
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(draft.DateNormalized)); err != nil {
		return fmt.Errorf("date must use YYYY-MM-DD")
	}
	if strings.TrimSpace(draft.CategoryNormalized) == "" {
		return fmt.Errorf("category is required")
	}
	if draft.HoursNormalized <= 0 {
		return fmt.Errorf("hours must be a positive number")
	}
	return nil
}
