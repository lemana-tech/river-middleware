package riverui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRiverUI_Middleware_HealthCheck(t *testing.T) {
	rc, pgPool := prep(t)

	// start a new savepoint so that the state of our test data stays
	// pristine between API calls
	tx, err := pgPool.Begin(t.Context())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tx.Rollback(t.Context())
	})

	opts := Options{
		RiverClient: rc,
		DevMode:     true,
		LiveFS:      false,
	}
	mw, err := NewMiddleware(t.Context(), opts)
	require.NoError(t, err)

	// create a dummy next handler
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
	})

	handler := mw.RiverUI(next)

	req := httptest.NewRequest(http.MethodGet, "/api/health-checks/minimal", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	t.Logf("Response body: %s", recorder.Body.String())

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.False(t, nextCalled)
	assert.NotEmpty(t, recorder.Body.String())
}

func TestRiverUI_Middleware_NonRiverUIPath(t *testing.T) {
	rc, pgPool := prep(t)

	tx, err := pgPool.Begin(t.Context())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tx.Rollback(t.Context())
	})

	opts := Options{
		RiverClient: rc,
		DevMode:     true,
		LiveFS:      false,
		BaseURL:     "/riverui",
	}
	mw, err := NewMiddleware(t.Context(), opts)
	require.NoError(t, err)

	// create a dummy next handler
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("next called"))
	})

	handler := mw.RiverUI(next)

	req := httptest.NewRequest("GET", "/not-riverui", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "next called", rw.Body.String())
}

func prep(t *testing.T) (*river.Client[pgx.Tx], *pgxpool.Pool) {
	pgC, err := postgres.Run(t.Context(),
		"postgres:18-alpine",
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		pgC.Terminate(t.Context())
	})

	pgURL, err := pgC.ConnectionString(t.Context())
	require.NoError(t, err)

	poolCfg, err := pgxpool.ParseConfig(pgURL)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(t.Context(), poolCfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	riverDriver := riverpgxv5.New(pool)

	migrator, err := rivermigrate.New(riverDriver, &rivermigrate.Config{})
	require.NoError(t, err)

	_, err = migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	client, err := river.NewClient(riverDriver, &river.Config{})
	require.NoError(t, err)

	return client, pool
}
