package ocrimport

import (
	"context"
	"testing"

	"invoice-app/internal/db"
)

type fakeStore struct {
	drafts      map[int64]db.OCRDraftEntry
	updated     map[int64]DraftEdit
	confirmedID []int64
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
func (f *fakeStore) GetSessionImages(sessionID int64) ([]db.OCRSessionImage, error) { return nil, nil }
func (f *fakeStore) SaveDraftEntries(sessionID int64, drafts []db.OCRDraftEntry) error {
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
func (f *fakeStore) ConfirmDraftEntries(sessionID int64, ids []int64) ([]int64, []int64, error) {
	f.confirmedID = ids
	return ids, nil, nil
}
func (f *fakeStore) DeleteOCRSession(id int64) error   { return nil }
func (f *fakeStore) ListCategories() ([]string, error) { return nil, nil }
func (f *fakeStore) ListRates() ([]db.Rate, error)     { return nil, nil }

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
