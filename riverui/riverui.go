package riverui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"riverqueue.com/riverui"
)

// Middleware wraps River UI handler into standard Go HTTP middleware
type Middleware struct {
	baseURL        string
	riveruiHandler http.Handler
}

// Option contains options for creating River UI middleware
type Options struct {
	// RiverClient is provided to initialize ui endpoints
	RiverClient *river.Client[pgx.Tx]

	// EndpointsTx is an optional transaction to wrap all database operations for api endpoints.
	// It's mainly used for testing.
	EndpointsTx *pgx.Tx

	// DevMode is whether the server is running in development mode
	DevMode bool

	// LiveFS is whether to use the live filesystem for the frontend
	LiveFS bool

	// Logger is the logger to use logging errors within the handler
	Logger *slog.Logger

	// BaseURL is the path for reverse proxy (e.g. "/riverui")
	BaseURL string
}

// NewMiddleware makes River UI middleware with given options
func NewMiddleware(ctx context.Context, opts Options) (*Middleware, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	baseURL := riverui.NormalizePathPrefix(opts.BaseURL)

	handler, err := riverui.NewHandler(&riverui.HandlerOpts{
		DevMode: opts.DevMode,
		Endpoints: riverui.NewEndpoints(opts.RiverClient, &riverui.EndpointsOpts[pgx.Tx]{
			Tx: opts.EndpointsTx,
		}),
		LiveFS: opts.LiveFS,
		Logger: opts.Logger,
		Prefix: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init riverui handler, %w", err)
	}

	if err := handler.Start(ctx); err != nil {
		return nil, fmt.Errorf("can't start riverui handler, %w", err)
	}

	return &Middleware{
		baseURL:        baseURL,
		riveruiHandler: handler,
	}, nil
}

// RiverUI middleware serves River UI endpoints on matching paths
func (m *Middleware) RiverUI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(strings.ToLower(r.URL.Path), m.baseURL) {
			m.riveruiHandler.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
