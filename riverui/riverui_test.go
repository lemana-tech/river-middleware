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
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
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
			name:           "non-riverui path with empty baseURL",
			baseURL:        "",
			requestPath:    "/not-riverui",
			method:         http.MethodGet,
			shouldCallNext: true,
			expectedCode:   http.StatusOK,
		},
		{
			name:           "uppercase path case insensitive",
			baseURL:        "",
			requestPath:    "/API/HEALTH-CHECKS/MINIMAL",
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
			requestPath:    "/api/health-checks/minimal",
			method:         http.MethodGet,
			shouldCallNext: true,
			expectedCode:   http.StatusOK,
		},
		{
			name:           "root path with empty baseURL",
			baseURL:        "",
			requestPath:    "/",
			method:         http.MethodGet,
			shouldCallNext: false,
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
		{
			name:           "POST method",
			baseURL:        "",
			requestPath:    "/api/queues",
			method:         http.MethodPost,
			shouldCallNext: false,
			expectedCode:   http.StatusOK,
		},
		{
			name:           "uppercase path with baseURL case insensitive",
			baseURL:        "/riverui",
			requestPath:    "/RIVERUI/API/HEALTH-CHECKS/MINIMAL",
			method:         http.MethodGet,
			shouldCallNext: false,
			expectedCode:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, pgPool := prep(t)

			tx, err := pgPool.Begin(t.Context())
			require.NoError(t, err)

			t.Cleanup(func() {
				_ = tx.Rollback(t.Context())
			})

			opts := Options{
				RiverClient: rc,
				EndpointsTx: &tx,
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

func prep(t *testing.T) (*river.Client[pgx.Tx], *pgxpool.Pool) {
	t.Helper()

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

	client, err := river.NewClient(riverDriver, &river.Config{
		Logger: riversharedtest.Logger(t),
	})
	require.NoError(t, err)

	return client, pool
}
