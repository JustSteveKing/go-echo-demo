# Go Echo Demo

[![CI](https://github.com/JustSteveKing/go-demo/actions/workflows/ci.yml/badge.svg)](https://github.com/JustSteveKing/go-demo/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/JustSteveKing/go-demo)](https://goreportcard.com/report/github.com/JustSteveKing/go-demo)
[![codecov](https://codecov.io/gh/JustSteveKing/go-demo/branch/main/graph/badge.svg)](https://codecov.io/gh/JustSteveKing/go-demo)
[![Release](https://img.shields.io/github/v/release/JustSteveKing/go-demo)](https://github.com/JustSteveKing/go-demo/releases)

A simple HTTP server built with Go using the Echo framework (labstack/echo) to demonstrate clean routing, built-in middleware, and graceful shutdown.

## Features

- ✨ Clean, organized code structure with Echo
- 🔄 Graceful shutdown handling
- 📝 Built-in request logging middleware
- 🛡️ Built-in panic recovery middleware
- 🧭 Simple routing with groups and middleware
- 🏥 Health check endpoint
- 📦 Version information endpoint

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

```bash
git clone <repository-url>
cd go-demo
go mod download
# Install Echo
go get github.com/labstack/echo/v4
go get github.com/labstack/echo/v4/middleware
```

### Running the Server

Using Make (recommended):
```bash
make run
```

Or for development (no binary build):
```bash
make dev
```

Or manually:
```bash
go run .
```

The server will start on `http://localhost:8080`

### Make Commands

This project includes a Makefile for common tasks. Run `make help` to see all available commands:

```bash
make help              # Show all available commands
make build             # Build the application
make test              # Run tests with coverage
make lint              # Run linter
make security          # Run security scanner
make ci                # Run all CI checks locally
make clean             # Clean build artifacts
make install-tools     # Install development tools
```

## API Endpoints

### `GET /`
Returns a simple greeting message.

```bash
curl http://localhost:8080/
```

### `GET /health`
Health check endpoint returning JSON status.

```bash
curl http://localhost:8080/health
```

Response:
```json
{"status":"ok"}
```

### `GET /version`
Returns build version information.

```bash
curl http://localhost:8080/version
```

Response:
```json
{"version":"dev","commit":"","built":""}
```

## Development with Echo

### Routing and Middleware (Echo)

This project uses Echo for HTTP routing and middleware. The equivalent server setup in Echo looks like this:

```go
import (
   "net/http"
   "github.com/labstack/echo/v4"
   emw "github.com/labstack/echo/v4/middleware"
)

func NewServer() *echo.Echo {
   e := echo.New()

   // Built-in middleware
   e.Use(emw.Recover())
   e.Use(emw.Logger())

   // Routes
   e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "Hello from Echo\n") })
   e.GET("/health", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok"}) })
   e.GET("/version", func(c echo.Context) error { return c.JSON(http.StatusOK, BuildInfo()) })

   return e
}
```

Echo also plays nicely with graceful shutdown via `Server.Shutdown(ctx)` when started with a custom http.Server.

### Building with Version Info

The Makefile automatically injects build information:

```bash
make build
```

Or build for all platforms:

```bash
make build-all
```

### Running CI Checks Locally

Before pushing, you can run the same checks that CI runs:

```bash
make ci
```

This will:
- Verify dependencies
- Run linter
- Run tests with race detection
- Run security scanner

### Installing Development Tools

```bash
make install-tools
```

This installs:
- golangci-lint (linting)
- gosec (security scanning)
- goreleaser (release testing)

## Project Structure

```
.
├── main.go        # Server setup and configuration (Echo bootstrap)
├── handlers.go    # HTTP request handlers (Echo context)
├── middleware.go  # HTTP middleware (Echo or custom)
├── types.go       # Type definitions and build info
└── go.mod         # Go module definition
```

## Configuration

Configuration is done via constants in `main.go` and by Echo middleware/server options:

- `serverPort` - Server listen address (default: `:8080`)
- `shutdownTimeout` - Graceful shutdown timeout (default: `10s`)
- `readTimeout` - HTTP read timeout (default: `5s`)
- `writeTimeout` - HTTP write timeout (default: `10s`)
- `idleTimeout` - HTTP idle timeout (default: `60s`)
- Echo middleware: `Recover`, `Logger`, plus optional CORS, GZIP, and request ID

## Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` signals for graceful shutdown. With Echo, you can run it on a custom `http.Server` and call `srv.Shutdown(ctx)` to allow in-flight requests to complete before stopping.

## CI/CD

This project uses GitHub Actions for continuous integration and deployment:

### CI Pipeline
- **Test**: Runs tests on Go 1.21, 1.22, and 1.23
- **Lint**: Code quality checks with golangci-lint
- **Security**: Security scanning with Gosec
- **Coverage**: Automated coverage reporting to Codecov

### Release Pipeline
To create a new release:
1. Create and push a new tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions will automatically build and release binaries for:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64, arm64)

## License

MIT
