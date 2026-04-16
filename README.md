# River middlewares

## RiverUI

### Overview

RiverUI middleware is an HTTP middleware wrapper for the river job queue ui dashboard. It integrates river ui endpoints into your Go HTTP server, allowing you to monitor and manage job queues through a web interface.

### Motivation

Middleware allows you to initialize [RiverUI](https://riverqueue.com/docs/river-ui) in an existing router with just a few lines of code. Regardless of where your application's router is initialized — whether in code you control or on the third-party side — you can use middleware for easy RiverUI integration.

### Basic setup

```go
import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-pkgz/routegroup"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lemana-tech/river-middleware/riverui"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
    pgPool, err := pgxpool.NewWithConfig(ctx, &pgxpool.Config{...})
    if err != nil {
        slog.Error("failed to init pgxpool", "err", err)
    }

    riverClient, err := river.NewClient(riverpgxv5.New(pgPool), nil)
    if err != nil {
        slog.Error("failed to init river client", "err", err)
    }

    mw, err := riverui.NewMiddleware(context.Background(), riverui.Options[pgx.Tx]{
        RiverClient:    riverClient,
        Logger:         slog.Default(),
        BaseURL:        "/riverui",
    })
    if err != nil {
        slog.Error("failed to init riverui middleware", "err", err)
    }

    router := routegroup.New(http.NewServeMux())
    router.Use(mw.RiverUI)
    http.ListenAndServe(":8080", router)
}
```

### Configuration options

| Option      | Type                | Description                                                                                      |
| ----------- | ------------------- | ------------------------------------------------------------------------------------------------ |
| RiverClient | \*river.Client[TTx] | River job queue client (required)                                                                |
| EndpointsTx | \*TTx               | Optional transaction to wrap all database operations for API endpoints (mainly used for testing) |
| DevMode     | bool                | Enable development mode                                                                          |
| LiveFS      | bool                | Use live filesystem for frontend assets                                                          |
| Logger      | \*slog.Logger       | Custom logger instance                                                                           |
| BaseURL     | string              | Base URL path for reverse proxy (e.g., `/riverui`)                                               |

`Options` and `NewMiddleware` are generic over `TTx` — the transaction type of your River driver (e.g., `pgx.Tx` for `riverpgxv5`, `*sql.Tx` for `riversqlite`).

### Access

Once running, access the River UI at: `http://localhost:8080/riverui`
