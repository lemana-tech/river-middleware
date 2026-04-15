package riverui

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // sqlite driver
)

func TestRiverUI_Middleware(t *testing.T) {
	type testCase struct {
		name           string
		baseURL        string
		requestPath    string
		method         string
		shouldCallNext bool
		expectedCode   int
	}

	tests := []testCase{
		{
			name:           "health check with empty baseURL",
			baseURL:        "",
			requestPath:    "/api/health-checks/minimal",
			method:         http.MethodGet,
			shouldCallNext: false,
			expectedCode:   http.StatusOK,
		},
		{
			name:           "matching baseURL path",
			baseURL:        "/riverui",
			requestPath:    "/riverui/api/health-checks/minimal",
			method:         http.MethodGet,
			shouldCallNext: false,
			expectedCode:   http.StatusOK,
		},
		{
			name:           "non-matching baseURL path",
			baseURL:        "/riverui",
			requestPath:    "/api/states",
			method:         http.MethodGet,
			shouldCallNext: true,
			expectedCode:   http.StatusOK,
		},
		{
			name:           "query parameters preserved",
			baseURL:        "",
			requestPath:    "/api/health-checks/minimal?test=1&foo=bar",
			method:         http.MethodGet,
			shouldCallNext: false,
			expectedCode:   http.StatusOK,
		},
	}

	riverClient, dbPool := prep(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tx, err := dbPool.Begin()
			require.NoError(t, err)

			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			opts := Options[*sql.Tx]{
				RiverClient: riverClient,
				DevMode:     true,
				LiveFS:      false,
				Logger:      riversharedtest.Logger(t),
				BaseURL:     tt.baseURL,
			}
			mw, err := NewMiddleware(t.Context(), opts)
			require.NoError(t, err)

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("next called"))
			})

			handler := mw.RiverUI(next)

			req := httptest.NewRequest(tt.method, tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.shouldCallNext, nextCalled, "next handler call mismatch")
			assert.Equal(t, tt.expectedCode, recorder.Result().StatusCode, "status code mismatch")
		})
	}
}

func prep(t *testing.T) (*river.Client[*sql.Tx], *sql.DB) {
	t.Helper()

	dbPool, err := sql.Open("sqlite", "file:./river.test")
	require.NoError(t, err)
	dbPool.SetMaxOpenConns(1)

	t.Cleanup(func() {
		dbPool.Close()
	})

	riverDriver := riversqlite.New(dbPool)

	migrator, err := rivermigrate.New(riverDriver, &rivermigrate.Config{})
	require.NoError(t, err)

	_, err = migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	client, err := river.NewClient(riverDriver, &river.Config{
		Logger: riversharedtest.Logger(t),
	})
	require.NoError(t, err)

	return client, dbPool
}
