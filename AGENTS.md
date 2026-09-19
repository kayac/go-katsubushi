# go-katsubushi Development Notes

## Project Architecture

**Core Service**: Unique ID generation service using Snowflake-like algorithm
- **Multi-protocol support**: memcached-compatible, HTTP API, gRPC
- **Auto worker ID assignment**: Redis-based distributed worker management
- **Graceful shutdown**: Context-based lifecycle management across all protocols

**Key Components**:
- `app.go` - Main application logic and custom slog handler
- `generator.go` - Snowflake-like ID generation algorithm
- `memcache.go`, `http.go`, `grpc.go` - Protocol implementations
- `cmd/katsubushi/main.go` - CLI entry point

## Build & Deployment

### Build Tools
- `make all` - Generate gRPC code and build binary
- `make test` - Run tests with race detection
- `make packages` - Cross-platform builds with GoReleaser
- `make docker` - Multi-architecture Docker builds
- `aqua` for CLI tool management
- `buf` for Protocol Buffers generation

### Deployment Options
- Standalone execution (manual worker ID)
- Redis-coordinated auto worker ID assignment
- Docker/Kubernetes deployment
- Unix Domain Socket support

## Development Practices

### Code Patterns
- **Context management**: Graceful shutdown across all servers
- **Atomic operations**: `sync/atomic` for statistics counters
- **Error handling**: Custom error types for domain-specific errors
- **Resource management**: `defer` for cleanup
- **Test injection**: `nowFunc` variable for time.Now substitution

### Logging (slog)
- Custom slog.Handler maintains original log format: `2025-08-01T22:11:28.043+0900    INFO    go-katsubushi/grpc.go:94        Message`
- Use `slog.SetDefault()` to avoid data races, not global logger variables
- Import `log/slog` in all files and use slog functions directly
- Debug logs added for all protocols (memcached, gRPC, HTTP API)

### Git Workflow
- Use individual file specification: `git add <file>` not `git add -A`
- Always run `go fmt ./...` before committing
- Commit messages and code comments in English

### Testing
- Run `go test -race ./...` to detect data races
- Protocol-specific test coverage (app_test.go, http_test.go, grpc_test.go, etc.)
- Worker ID uniqueness validation
- Build verification: `go build ./cmd/katsubushi/`

### Dependencies
- **Core**: gRPC, Protocol Buffers, gomemcache
- **Infrastructure**: fujiwara/raus (Redis auto-assignment)
- **Development**: buf, aqua, goreleaser
