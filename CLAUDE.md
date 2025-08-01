# go-katsubushi Development Notes

## Logging (slog)

- Custom slog.Handler maintains original log format: `2025-08-01T22:11:28.043+0900    INFO    go-katsubushi/grpc.go:94        Message`
- Use `slog.SetDefault()` to avoid data races, not global logger variables
- Import `log/slog` in all files and use slog functions directly
- Debug logs added for all protocols (memcached, gRPC, HTTP API)

## Development Practices

### Git Workflow
- Use individual file specification: `git add <file>` not `git add -A`
- Always run `go fmt ./...` before committing
- Commit messages and code comments in English

### Testing
- Run `go test -race ./...` to detect data races
- Build verification: `go build ./cmd/katsubushi/`
- Manual testing with different protocols and log levels

### Code Quality
- Follow existing code patterns and conventions
- Preserve original functionality while implementing changes
- Use structured logging with key-value pairs for better observability
