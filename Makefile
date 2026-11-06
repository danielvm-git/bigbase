.PHONY: all build test test-scripts lint preflight mcp mcp-check clean

all: build

build:
	go build -o bigbase .

test:
	go test ./...

# Gate-script tests (bash gates driven from Node's test runner).
test-scripts:
	node --test scripts/*.test.cjs

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
