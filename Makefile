# BigBase — thin task aliases over the existing go/npm commands.
.PHONY: build test lint preflight setup gen-mcp clean

build:          ## Build the single binary
	go build -o bigbase .

test:           ## Run the Go test suite
	go test ./... -count=1

lint:           ## Run golangci-lint
	golangci-lint run ./...

preflight:      ## Full preflight (vet, test, gosec, build, secrets)
	npm run preflight

setup:          ## One-command dev onboarding
	./scripts/setup.sh

gen-mcp:        ## Regenerate MCP adapter configs from canonical .mcp.json
	./scripts/gen-mcp-configs.sh

clean:          ## Remove build/test artifacts (rebuildable)
	rm -f bigbase bigbase-linux-amd64 bigbase_linux deploy.test coverage.out
	rm -f *.db *.db-shm *.db-wal
