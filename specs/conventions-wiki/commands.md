# Commands

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

## Commands
| Action | Command |
|--------|---------|
| Run    | `go run .` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Preflight | `npm run preflight` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |
