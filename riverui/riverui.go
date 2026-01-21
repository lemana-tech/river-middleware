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
	riveruiHandler http.Handler
	prefix         string
}

// Option contains options for creating River UI middleware
type Options struct {
	// RiverClient is provided to initialize ui endpoints
	RiverClient *river.Client[*pgx.Tx]

	// DevMode is whether the server is running in development mode
	DevMode bool

	// LiveFS is whether to use the live filesystem for the frontend
	LiveFS bool

	// Logger is the logger to use logging errors within the handler
	Logger *slog.Logger

	// Prefix is the path prefix to use for the API and UI HTTP requests
	Prefix string
}

// NewMiddleware makes River UI middleware with given options
func NewMiddleware(ctx context.Context, opts Options) (*Middleware, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	prefix := riverui.NormalizePathPrefix(opts.Prefix)

	handler, err := riverui.NewHandler(&riverui.HandlerOpts{
		DevMode:   opts.DevMode,
		Endpoints: riverui.NewEndpoints(opts.RiverClient, nil),
		LiveFS:    opts.LiveFS,
		Logger:    opts.Logger,
		Prefix:    prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init riverui handler, %w", err)
	}

	if err := handler.Start(ctx); err != nil {
		return nil, fmt.Errorf("can't start riverui handler, %w", err)
	}

	return &Middleware{
		riveruiHandler: handler,
		prefix:         prefix,
	}, nil
}

// RiverUI middleware serves River UI endpoints on given prefix
func (m *Middleware) RiverUI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, m.prefix) {
			m.riveruiHandler.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
