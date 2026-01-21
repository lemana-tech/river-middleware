# River middlewares

## RiverUI

### Overview

RiverUI middleware is an HTTP middleware wrapper for the river job queue ui dashboard. It integrates river ui endpoints into your Go HTTP server, allowing you to monitor and manage job queues through a web interface.

### Purpose

This middleware enables:

- Real-time job queue monitoring and management
- Seamless integration with existing Go HTTP servers
- Customizable UI path prefix
- Dev and LiveFS modes support

### Basic Setup

```go
import (
	"context"
	"log/slog"
    "net/http"

	"github.com/jackc/pgx/v5"
    "github.com/go-pkgz/routegroup"
	"github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/lemana-tech/river-middleware"
)

func main() {
    riverClient, err := river.NewClient(riverpgxv5.New(pgPool), nil)
    if err != nil {
        slog.Error("failed to init river client", "err", err)
    }

    mw, err := riverui.NewMiddleware(context.Background(), riverui.Options{
        RiverClient: riverClient,
        DevMode:     false,
        LiveFS:      false,
        Logger:      slog.Default(),
        Prefix:      "/riverui",
    })
    if err != nil {
        slog.Error("failed to init riverui middleware", "err", err)
    }

    router := routegroup.New(http.NewServeMux())
    router.Use(mw.RiverUI)
    http.ListenAndServe(":8080", router)
}
```

### Configuration Options

| Option      | Type           | Description                                     |
| ----------- | -------------- | ----------------------------------------------- |
| RiverClient | \*river.Client | River job queue client (required)               |
| DevMode     | bool           | Enable development mode                         |
| LiveFS      | bool           | Use live filesystem for frontend assets         |
| Logger      | \*slog.Logger  | Custom logger instance                          |
| Prefix      | string         | URL path prefix for River UI (e.g., `/riverui`) |

### Access

Once running, access the River UI at: `http://localhost:8080/riverui`
