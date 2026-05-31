package ocrimport

import (
	"context"
	"testing"

	"invoice-app/internal/db"
	"invoice-app/internal/ocr"
)

type fakeStore struct {
	drafts      map[int64]db.OCRDraftEntry
	updated     map[int64]DraftEdit
	confirmedID []int64
	images      []db.OCRSessionImage
	categories  []string
	rates       []db.Rate
	savedDrafts []db.OCRDraftEntry
}

func (f *fakeStore) CreateOCRSession() (int64, error) { return 1, nil }
func (f *fakeStore) GetOCRSession(id int64) (db.OCRSession, error) {
	return db.OCRSession{ID: id}, nil
}
func (f *fakeStore) UpdateOCRSessionState(id int64, state string, errorMsg string) error { return nil }
func (f *fakeStore) ListOCRSessions() ([]db.OCRSession, error)                           { return nil, nil }
func (f *fakeStore) AddSessionImage(sessionID int64, fileName, filePath string, fileSize int64) error {
	return nil
}
func (f *fakeStore) GetSessionImages(sessionID int64) ([]db.OCRSessionImage, error) {
	return f.images, nil
}
func (f *fakeStore) SaveDraftEntries(sessionID int64, drafts []db.OCRDraftEntry) error {
	f.savedDrafts = drafts
	return nil
}
func (f *fakeStore) GetDraftEntries(sessionID int64) ([]db.OCRDraftEntry, error) { return nil, nil }
func (f *fakeStore) GetDraftEntry(id int64) (db.OCRDraftEntry, error) {
	return f.drafts[id], nil
}
func (f *fakeStore) UpdateDraftEntry(id int64, date, category, hours, notes string) error {
	if f.updated == nil {
		f.updated = make(map[int64]DraftEdit)
	}
	f.updated[id] = DraftEdit{Date: date, Category: category, Hours: hours, Notes: notes}
	d := f.drafts[id]
	d.DateNormalized = date
	d.CategoryNormalized = category
	d.HoursNormalized = 1
	f.drafts[id] = d
	return nil
}
func (f *fakeStore) DeleteDraftEntry(sessionID, draftID int64) error {
	delete(f.drafts, draftID)
	return nil
}
func (f *fakeStore) ConfirmDraftEntries(sessionID int64, ids []int64) ([]int64, []int64, error) {
	f.confirmedID = ids
	return ids, nil, nil
}
func (f *fakeStore) DeleteOCRSession(id int64) error   { return nil }
func (f *fakeStore) ListCategories() ([]string, error) { return f.categories, nil }
func (f *fakeStore) ListRates() ([]db.Rate, error)     { return f.rates, nil }

type fakeExtractor struct {
	calls [][]string
}

func (f *fakeExtractor) Extract(ctx context.Context, images []string, hints ocr.ContextHint, rates []ocr.RateHint, sessionID int64) (*ocr.OCRResponse, error) {
	f.calls = append(f.calls, append([]string(nil), images...))
	return &ocr.OCRResponse{
		SessionID: sessionID,
		Status:    "success",
		Entries: []ocr.OCRExtractedEntry{
			{
				Date:     ocr.OCRExtractedField{Raw: "1/5", Normalized: "2026-05-01", Confidence: 0.8},
				Category: ocr.OCRExtractedField{Raw: "Consulting", Normalized: "Consulting", Confidence: 0.9},
				Hours:    ocr.OCRExtractedField{Raw: "2", Normalized: "2", Confidence: 0.9},
				Notes:    ocr.OCRExtractedField{Raw: "page", Normalized: "page", Confidence: 0.7},
			},
		},
		Metadata: ocr.OCRMetadata{ModelUsed: "test", PagesProcessed: len(images), AvgConfidence: 0.9},
	}, nil
}

func TestProcessSessionExtractsOneImagePerRequest(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		images: []db.OCRSessionImage{
			{ID: 1, SessionID: 10, FilePath: "/tmp/page-1.jpg"},
			{ID: 2, SessionID: 10, FilePath: "/tmp/page-2.heic"},
		},
		categories: []string{"Consulting"},
	}
	extractor := &fakeExtractor{}
	importer := Importer{Store: store, Extractor: extractor}

	if err := importer.ProcessSession(context.Background(), 10); err != nil {
		t.Fatalf("ProcessSession returned error: %v", err)
	}

	if len(extractor.calls) != 2 {
		t.Fatalf("Extract call count = %d, want 2", len(extractor.calls))
	}
	if len(extractor.calls[0]) != 1 || extractor.calls[0][0] != "/tmp/page-1.jpg" {
		t.Fatalf("first Extract images = %#v", extractor.calls[0])
	}
	if len(extractor.calls[1]) != 1 || extractor.calls[1][0] != "/tmp/page-2.heic" {
		t.Fatalf("second Extract images = %#v", extractor.calls[1])
	}
	if len(store.savedDrafts) != 2 {
		t.Fatalf("saved draft count = %d, want 2", len(store.savedDrafts))
	}
}

func TestDeleteDraftRejectsDraftFromAnotherSession(t *testing.T) {
	t.Parallel()

	store := &fakeStore{drafts: map[int64]db.OCRDraftEntry{
		1: {ID: 1, SessionID: 20, DateNormalized: "2026-05-23", CategoryNormalized: "Consulting", HoursNormalized: 1},
	}}
	importer := Importer{Store: store}

	err := importer.DeleteDraft(context.Background(), 10, 1)
	if err == nil {
		t.Fatal("DeleteDraft returned nil error for another session's draft")
	}
	if _, exists := store.drafts[1]; !exists {
		t.Fatal("DeleteDraft removed another session's draft")
	}
}

func TestDeleteDraftRemovesUnconfirmedDraft(t *testing.T) {
	t.Parallel()

	store := &fakeStore{drafts: map[int64]db.OCRDraftEntry{
		1: {ID: 1, SessionID: 10, DateNormalized: "2026-05-23", CategoryNormalized: "Consulting", HoursNormalized: 1},
	}}
	importer := Importer{Store: store}

	if err := importer.DeleteDraft(context.Background(), 10, 1); err != nil {
		t.Fatalf("DeleteDraft returned error: %v", err)
	}
	if _, exists := store.drafts[1]; exists {
		t.Fatal("DeleteDraft left draft in store")
	}
}

func TestConfirmDraftsValidatesExistingDrafts(t *testing.T) {
	t.Parallel()

	store := &fakeStore{drafts: map[int64]db.OCRDraftEntry{
		1: {ID: 1, SessionID: 10, DateNormalized: "not-a-date", CategoryNormalized: "Consulting", HoursNormalized: 1},
	}}
	importer := Importer{Store: store}

	_, err := importer.ConfirmDrafts(context.Background(), ConfirmDraftsInput{SessionID: 10, IDs: []int64{1}})
	if err == nil {
		t.Fatal("ConfirmDrafts returned nil error for invalid date")
	}
}

func TestConfirmDraftsAppliesAndValidatesEdits(t *testing.T) {
	t.Parallel()

	store := &fakeStore{drafts: map[int64]db.OCRDraftEntry{
		1: {ID: 1, SessionID: 10, DateNormalized: "", CategoryNormalized: "", HoursNormalized: 0},
	}}
	importer := Importer{Store: store}

	got, err := importer.ConfirmDrafts(context.Background(), ConfirmDraftsInput{
		SessionID: 10,
		IDs:       []int64{1},
		Edits: map[int64]DraftEdit{
			1: {Date: "2026-05-23", Category: "Consulting", Hours: "2.5", Notes: "reviewed"},
		},
	})
	if err != nil {
		t.Fatalf("ConfirmDrafts returned error: %v", err)
	}
	if len(got.Confirmed) != 1 || got.Confirmed[0] != 1 {
		t.Fatalf("Confirmed = %#v, want [1]", got.Confirmed)
	}
	if store.updated[1].Category != "Consulting" {
		t.Fatalf("updated category = %q, want Consulting", store.updated[1].Category)
	}
}
