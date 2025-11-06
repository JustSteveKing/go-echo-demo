# Setup Guide: Building a Production-Ready Go HTTP Server with Echo

This guide walks you through creating this Go HTTP server using the Echo framework from scratch, explaining each component and the architecture decisions behind it.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Project Initialization](#project-initialization)
3. [Core Application Structure](#core-application-structure)
4. [Install Echo](#install-echo)
5. [Building the HTTP Server (Echo)](#building-the-http-server-echo)
6. [Adding Middleware (Echo)](#adding-middleware-echo)
7. [Implementing Handlers (Echo)](#implementing-handlers-echo)
7. [Type Definitions](#type-definitions)
8. [Build Automation with Make](#build-automation-with-make)
9. [CI/CD Setup](#cicd-setup)
10. [Testing the Application](#testing-the-application)

---

## Prerequisites

### Required Software

- **Go**: Version 1.21 or higher
  ```bash
  # Check your Go version
  go version
  
  # Install Go (macOS)
  brew install go
  
  # Install Go (Linux)
  wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
  ```

- **Git**: For version control
  ```bash
  git --version
  ```

### Optional Development Tools

These will be installed via `make install-tools`:
- `golangci-lint` - Comprehensive linting
- `gosec` - Security scanning
- `goreleaser` - Release automation

---

## Project Initialization

### Step 1: Create Project Directory

```bash
mkdir go-demo
cd go-demo
```

### Step 2: Initialize Go Module

```bash
go mod init github.com/yourusername/go-demo
```

This creates a `go.mod` file:

```go
module github.com/yourusername/go-demo

go 1.23.4
```

**Why?** Go modules provide dependency management and versioning. The module path should match your repository URL.

---

## Core Application Structure

The project follows a simple, flat structure suitable for small-to-medium applications:

```
go-demo/
├── main.go           # Server setup, configuration, and lifecycle
├── handlers.go       # HTTP request handlers
├── middleware.go     # HTTP middleware functions
├── types.go          # Type definitions and build metadata
├── go.mod            # Go module definition
├── Makefile          # Build automation
├── .goreleaser.yml   # Release configuration
└── .github/
    └── workflows/
        ├── ci.yml        # Continuous integration
        └── release.yml   # Release automation
```

**Design Philosophy:**
- **Single package**: For small projects, avoiding premature abstraction
- **Separation of concerns**: Different files for different responsibilities
- **Lean dependencies**: Echo for routing/middleware; stdlib for the rest
- **Production-ready**: Graceful shutdown, timeouts, and middleware

---

## Install Echo

Add Echo and its middleware package:

```bash
go get github.com/labstack/echo/v4
go get github.com/labstack/echo/v4/middleware
```

---

## Building the HTTP Server (Echo)

### Step 3: Create `types.go`

Start with type definitions and build metadata:

```go
package main

import "net/http"

// Build info set via -ldflags at build time (optional).
var (
	buildVersion = "dev"
	buildCommit  = ""
	buildTime    = ""
)

// Info holds build metadata for the binary.
type Info struct {
	Version string
	Commit  string
	Built   string
}

// BuildInfo returns the build metadata.
func BuildInfo() Info {
	return Info{
		Version: buildVersion,
		Commit:  buildCommit,
		Built:   buildTime,
	}
}

// responseWriter wraps http.ResponseWriter to record the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the status code and writes the header.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
```

**Key Concepts:**
- **Build variables**: Injected at compile time for version tracking
- **responseWriter**: Custom wrapper to capture HTTP status codes for logging
- **Public functions**: `BuildInfo()` provides structured access to build metadata

### Step 4: Create `main.go`

Bootstrap Echo with graceful shutdown:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

  "github.com/labstack/echo/v4"
  emw "github.com/labstack/echo/v4/middleware"
)

const (
	serverPort        = ":8080"
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 2 * time.Second
	readTimeout       = 5 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 1 << 20 // 1MB
)

var newRunContext = func() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func main() {
	ctx, cancel := newRunContext()
	defer cancel()

  // Echo instance
  e := echo.New()

  // Built-in middleware
  e.Use(emw.Recover())
  e.Use(emw.Logger())

  // Routes
  e.GET("/", hello)
  e.GET("/health", health)
  e.GET("/version", version)

  // Use custom HTTP server to configure timeouts
  srv := &http.Server{
    Addr:              serverPort,
    Handler:           e,
    ReadHeaderTimeout: readHeaderTimeout,
    ReadTimeout:       readTimeout,
    WriteTimeout:      writeTimeout,
    IdleTimeout:       idleTimeout,
    MaxHeaderBytes:    maxHeaderBytes,
  }

  // Start server in background
  go func() {
    log.Printf("Listening on %s", serverPort)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
      log.Fatalf("server error: %v", err)
    }
  }()

  // Wait for shutdown signal
  <-ctx.Done()
  log.Println("shutdown signal received")

  // Graceful shutdown
  shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
  defer shutdownCancel()

  if err := srv.Shutdown(shutdownCtx); err != nil {
    log.Printf("graceful shutdown failed: %v", err)
  }
  log.Println("server stopped")
}
```

**Architecture Decisions:**

1. **Constants for Configuration**: All timeouts and limits defined upfront
   - `ReadHeaderTimeout`: Prevents slow-read attacks
   - `ReadTimeout`: Maximum time to read entire request
   - `WriteTimeout`: Maximum time to write response
   - `IdleTimeout`: Maximum idle time for keep-alive connections
   - `maxHeaderBytes`: Limits request header size (1MB)

2. **Graceful Shutdown**: 
   - Uses `signal.NotifyContext` to catch SIGINT/SIGTERM
   - Gives in-flight requests time to complete (10s timeout)
   - Critical for zero-downtime deployments

3. **Context Management**:
   - Application context tied to OS signals
   - Separate shutdown context with timeout
   - Follows Go best practices for cancellation

4. **Server in Goroutine**: Allows main goroutine to wait for shutdown signals

5. **Middleware**: Echo's `Recover` must wrap all handlers; `Logger` provides request logs

---

## Adding Middleware (Echo)

### Step 5: Create `middleware.go`

Middleware functions wrap Echo handlers to add cross-cutting concerns. Echo provides many out-of-the-box middlewares. You can still add custom ones if needed:

```go
package main

import (
	"log"
	"time"

  "github.com/labstack/echo/v4"
)

// exampleCustomMiddleware shows how to write a simple Echo middleware
func exampleCustomMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
  return func(c echo.Context) error {
    start := time.Now()
    err := next(c)
    dur := time.Since(start)
    log.Printf("%s %s %d %s", c.Request().Method, c.Path(), c.Response().Status, dur)
    return err
  }
}
```

**Middleware Patterns:**

1. **Logging Middleware (Echo)**:
  - Use Echo's built-in `middleware.Logger()` for request logging
  - Or write a custom middleware as shown above

2. **Recovery Middleware**:
  - Use Echo's built-in `middleware.Recover()` to catch panics
  - Should be outermost middleware

3. **Composition**:
  - Echo composes middleware with `e.Use(...)` and per-route/per-group middleware

**Order Matters**: Recovery must wrap everything to catch all panics.

---

## Implementing Handlers (Echo)

### Step 6: Create `handlers.go`

HTTP handlers respond to specific routes using Echo's context:

```go
package main

import (
  "net/http"
  "github.com/labstack/echo/v4"
)

func hello(c echo.Context) error {
  return c.String(http.StatusOK, "Hello from Echo\n")
}

func health(c echo.Context) error {
  return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func version(c echo.Context) error {
  info := BuildInfo()
  return c.JSON(http.StatusOK, info)
}
```

**Handler Best Practices:**

1. **Hello Handler** (`/`):
  - Simple text response
  - Echo sets appropriate headers via helpers
  - Good for basic health checks and testing

2. **Health Handler** (`/health`):
  - JSON response for monitoring systems
  - Returns 200 OK with status
  - Essential for Kubernetes/Docker health checks
  - Could be extended to check database connectivity, etc.

3. **Version Handler** (`/version`):
  - Exposes build metadata as JSON
  - Uses build variables set at compile time
  - Helpful for debugging deployed versions
  - Can verify correct version in production

**Error Handling**: Echo helpers return errors to be handled by Echo.

---

## Build Automation with Make

### Step 7: Create `Makefile`

A comprehensive Makefile for all development tasks:

```makefile
.PHONY: help build test lint security coverage clean run install-tools all check

# Variables
BINARY_NAME=go-demo
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.buildVersion=$(VERSION) -X main.buildCommit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

all: clean install-tools check build ## Run all checks and build

check: lint test security ## Run all checks (lint, test, security)

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/goreleaser/goreleaser@latest
	@echo "Tools installed successfully!"

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Multi-platform build complete!"

run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

dev: ## Run the application without building binary
	$(GORUN) .

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Tests complete!"

test-short: ## Run tests without race detector (faster)
	@echo "Running tests (short)..."
	$(GOTEST) -v -short ./...

coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-tools'" && exit 1)
	golangci-lint run --timeout=5m ./...
	@echo "Linting complete!"

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .
	@echo "Code formatted!"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...
	@echo "Vet complete!"

security: ## Run security scanner
	@echo "Running security scanner..."
	@which gosec > /dev/null || (echo "gosec not found. Run 'make install-tools'" && exit 1)
	gosec -quiet ./...
	@echo "Security scan complete!"

mod-download: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded!"

mod-tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies tidied!"

mod-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GOMOD) verify
	@echo "Dependencies verified!"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf bin/
	@rm -f coverage.txt coverage.html
	@echo "Clean complete!"

release-test: ## Test release build locally
	@echo "Testing release build..."
	@which goreleaser > /dev/null || (echo "goreleaser not found. Run 'make install-tools'" && exit 1)
	goreleaser release --snapshot --clean --skip=publish
	@echo "Release test complete!"

version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

ci: mod-verify lint test security ## Run CI checks locally

.DEFAULT_GOAL := help
```

**Makefile Features:**

1. **Self-Documenting**: `make help` shows all commands
2. **Build Version Injection**: Uses `-ldflags` to set version info
3. **Cross-Platform Builds**: `make build-all` creates binaries for all platforms
4. **CI Simulation**: `make ci` runs the same checks as GitHub Actions
5. **Tool Installation**: `make install-tools` sets up development environment
6. **Code Quality**: Lint, test, security scan, format
7. **Coverage Reports**: Generate HTML coverage reports

**Usage:**
```bash
make help          # See all commands
make install-tools # First-time setup
make dev          # Quick development
make ci           # Pre-commit checks
make build        # Production build
```

---

## CI/CD Setup

### Step 8: GitHub Actions Workflows

#### Continuous Integration (`.github/workflows/ci.yml`)

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.21', '1.22', '1.23']
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go-version }}

    - name: Cache Go modules
      uses: actions/cache@v4
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Download dependencies
      run: go mod download

    - name: Verify dependencies
      run: go mod verify

    - name: Build
      run: go build -v ./...

    - name: Run tests
      run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage.txt
        flags: unittests
        name: codecov-umbrella

  lint:
    name: Lint
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'

    - name: golangci-lint
      uses: golangci/golangci-lint-action@v6
      with:
        version: latest
        args: --timeout=5m

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'

    - name: Run Gosec Security Scanner
      uses: securego/gosec@master
      with:
        args: './...'
```

**CI Pipeline Features:**
- **Matrix Testing**: Tests across Go 1.21, 1.22, and 1.23
- **Dependency Caching**: Speeds up builds
- **Race Detection**: `-race` flag catches concurrency issues
- **Coverage Reporting**: Automatic upload to Codecov
- **Parallel Jobs**: Test, lint, and security scan run concurrently

#### Release Automation (`.github/workflows/release.yml`)

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    name: Create Release
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'

    - name: Run GoReleaser
      uses: goreleaser/goreleaser-action@v6
      with:
        distribution: goreleaser
        version: latest
        args: release --clean
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Step 9: GoReleaser Configuration (`.goreleaser.yml`)

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go mod verify

builds:
  - id: go-demo
    binary: go-demo
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.buildVersion={{.Version}}
      - -X main.buildCommit={{.Commit}}
      - -X main.buildTime={{.Date}}
    mod_timestamp: '{{ .CommitTimestamp }}'

archives:
  - id: go-demo
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
      - '^ci:'
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: 'Bug fixes'
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: 'Performance improvements'
      regexp: '^.*?perf(\([[:word:]]+\))??!?:.+$'
      order: 2
    - title: Others
      order: 999

release:
  github:
    owner: yourusername
    name: go-demo
  draft: false
  prerelease: auto
  name_template: "{{.ProjectName}} v{{.Version}}"
```

**Release Features:**
- **Multi-Platform**: Builds for Linux, macOS, Windows (amd64 + arm64)
- **Static Binaries**: `CGO_ENABLED=0` for portability
- **Version Injection**: Build info embedded via ldflags
- **Archives**: Compressed releases with README and LICENSE
- **Changelog Generation**: Automatic from commit messages
- **GitHub Integration**: Creates releases automatically on tag push

**Creating a Release:**
```bash
# Tag and push
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions will automatically:
# 1. Build binaries for all platforms
# 2. Create GitHub release
# 3. Upload binaries
# 4. Generate changelog
```

---

## Testing the Application

### Step 10: Run the Application

```bash
# Install development tools
make install-tools

# Run all CI checks locally
make ci

# Run in development mode
make dev
```

The server should start on `http://localhost:8080`.

### Step 11: Test Endpoints

Open a new terminal and test:

```bash
# Test hello endpoint
curl http://localhost:8080/
# Expected: Hello from Sevalla

# Test health endpoint
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Test version endpoint
curl http://localhost:8080/version
# Expected: {"version":"dev","commit":"","built":""}
```

### Step 12: Test Graceful Shutdown

In the server terminal, press `Ctrl+C`. You should see:
```
shutdown signal received
server stopped
```

The server waits for in-flight requests to complete before stopping.

---

## Production Deployment Checklist

### Environment Configuration

1. **Environment Variables** (if you add them):
   ```bash
   export SERVER_PORT=8080
   export LOG_LEVEL=info
   ```

2. **Health Check Endpoint**: `/health` for load balancers

3. **Metrics** (future enhancement):
   - Consider adding Prometheus metrics
   - Track request counts, durations, errors

### Build for Production

```bash
# Build optimized binary
make build

# Or build for specific platform
GOOS=linux GOARCH=amd64 make build

# Test the binary
./go-demo
```

### Container Deployment (Optional)

Create a `Dockerfile`:

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o go-demo .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/go-demo .
EXPOSE 8080
CMD ["./go-demo"]
```

Build and run:
```bash
docker build -t go-demo .
docker run -p 8080:8080 go-demo
```

### Kubernetes Deployment (Optional)

Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-demo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: go-demo
  template:
    metadata:
      labels:
        app: go-demo
    spec:
      containers:
      - name: go-demo
        image: go-demo:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

---

## Advanced Enhancements

### Add Database Connection

```go
// In main.go
db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Pass db to handlers that need it
```

### Add Configuration Management

```go
type Config struct {
    Port         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    // ...
}

func loadConfig() Config {
    return Config{
        Port: getEnv("PORT", "8080"),
        // ...
    }
}
```

### Add Structured Logging

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("server started", "port", serverPort)
```

### Add Request Validation

```go
func validateRequest(r *http.Request) error {
    if r.Method != http.MethodPost {
        return fmt.Errorf("method not allowed")
    }
    // More validation...
    return nil
}
```

### Add Rate Limiting

```go
import "golang.org/x/time/rate"

func rateLimitMiddleware(limiter *rate.Limiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Find process using port 8080
   lsof -i :8080
   # Kill the process
   kill -9 <PID>
   ```

2. **Module Import Errors**
   ```bash
   go mod tidy
   go mod verify
   ```

3. **Build Fails**
   ```bash
   # Clean and rebuild
   make clean
   make build
   ```

4. **Tests Fail**
   ```bash
   # Run tests with verbose output
   go test -v ./...
   ```

---

## Summary

You now have a production-ready Go HTTP server with:

✅ **Clean Architecture**: Organized, maintainable code structure  
✅ **Production Safety**: Graceful shutdown, timeouts, panic recovery  
✅ **Observability**: Request logging and version endpoints  
✅ **Build Automation**: Comprehensive Makefile for all tasks  
✅ **CI/CD**: Automated testing, linting, and releases  
✅ **Cross-Platform**: Builds for Linux, macOS, Windows  
✅ **Security**: Automated security scanning  
✅ **Quality**: Linting, testing, coverage reporting  

### Next Steps

1. **Add Tests**: Write unit tests for handlers and middleware
2. **Add Metrics**: Integrate Prometheus for monitoring
3. **Add Tracing**: Implement OpenTelemetry for distributed tracing
4. **Add Database**: Connect to PostgreSQL/MySQL
5. **Add Authentication**: Implement JWT or OAuth2
6. **Add API Documentation**: Generate OpenAPI/Swagger docs

### Resources

- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Library](https://pkg.go.dev/std)
- [golangci-lint](https://golangci-lint.run/)
- [GoReleaser](https://goreleaser.com/)

---

**Happy Coding! 🚀**
