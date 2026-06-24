#!/bin/bash
# New Relic MCP wrapper — injects API key from environment
exec /Users/danielvm/Developer/bigbase/node_modules/.bin/mcp-remote \
  "https://mcp.newrelic.com/mcp/" \
  --header "Api-Key: ${NEW_RELIC_API_KEY:?NEW_RELIC_API_KEY not set}"
