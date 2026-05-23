package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func newTestServer(t *testing.T) (*httptest.Server, *db.Store) {
	t.Helper()
	store := testutil.NewTestDB(t)
	cfg := config.Config{
		Address:     ":8080",
		AuthEnabled: false,
	}
	logger := NewLogger("error", "text", false)

	migDir := testutil.MigrationsDir(t)
	multiStore := db.NewMultiStore(t.TempDir(), migDir)
	multiStore.SetLegacyStore(store)

	app := New(multiStore, nil, nil, nil, cfg, logger)
	app.SetLegacyStore(store)

	server := httptest.NewServer(app.Routes())
	t.Cleanup(server.Close)

	return server, store
}

func TestIntegrationDashboardReturns200(t *testing.T) {
	t.Parallel()
	server, store := newTestServer(t)

	require.NoError(t, store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, store.SaveSettings(testutil.SampleSettings()))

	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Dashboard")
}

func TestIntegrationStaticFilesServed(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t)

	resp, err := http.Get(server.URL + "/static/styles.css")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegrationEntriesPageReturns200(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t)

	resp, err := http.Get(server.URL + "/entries")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegrationHealthReturns200(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t)

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegrationSettingsPageReturns200(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t)

	resp, err := http.Get(server.URL + "/settings")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
