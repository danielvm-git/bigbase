# Commands

source: AGENTS.md
references: [AGENTS.md]

## Commands
| Action | Command |
|--------|---------|
| Run    | `go run .` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Preflight | `npm run preflight` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |
