package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteDraftEntryRemovesOnlyRequestedDraft(t *testing.T) {
	t.Parallel()

	store := setupTestDB(t)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "review_ready", ""))
	require.NoError(t, store.SaveDraftEntries(sessionID, []OCRDraftEntry{
		{
			DateNormalized:     "2026-05-24",
			CategoryNormalized: "Consulting",
			HoursNormalized:    2,
			Confidence:         0.8,
		},
		{
			DateNormalized:     "2026-05-25",
			CategoryNormalized: "Admin",
			HoursNormalized:    1,
			Confidence:         0.7,
		},
	}))
	drafts, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, drafts, 2)

	require.NoError(t, store.DeleteDraftEntry(sessionID, drafts[0].ID))

	remaining, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, drafts[1].ID, remaining[0].ID)
	session, err := store.GetOCRSession(sessionID)
	require.NoError(t, err)
	assert.Equal(t, "review_ready", session.State)
}

func TestDeleteDraftEntryRejectsConfirmedDraft(t *testing.T) {
	t.Parallel()

	store := setupTestDB(t)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "review_ready", ""))
	require.NoError(t, store.SaveDraftEntries(sessionID, []OCRDraftEntry{
		{
			DateNormalized:     "2026-05-24",
			CategoryNormalized: "Consulting",
			HoursNormalized:    2,
			Confidence:         0.8,
		},
	}))
	drafts, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, drafts, 1)
	_, _, err = store.ConfirmDraftEntries(sessionID, []int64{drafts[0].ID})
	require.NoError(t, err)

	err = store.DeleteDraftEntry(sessionID, drafts[0].ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	remaining, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.True(t, remaining[0].Confirmed)
}
