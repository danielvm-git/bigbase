.PHONY: all build test lint preflight mcp mcp-check clean

all: build

build:
	go build -o bigbase .

test:
	go test ./...

lint:
	golangci-lint run ./...

preflight:
	npm run preflight

# MCP adapters are generated from the canonical .mcp.json — edit that file,
# then regenerate. `make mcp-check` is what CI runs.
mcp:
	bash scripts/gen-mcp-configs.sh

mcp-check:
	bash scripts/gen-mcp-configs.sh --check

clean:
	rm -f bigbase bigbase-linux-amd64 bigbase-darwin-* bigbase_linux *.test *.out coverage.out
