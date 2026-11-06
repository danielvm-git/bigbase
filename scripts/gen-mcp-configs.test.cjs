'use strict';

// Story: e90s05 — .mcp.json is the single canonical MCP source and
// opencode.jsonc is generated from it. Pins two contracts:
//   1. the generator's --check mode stays in sync with the committed adapter,
//   2. the `make mcp` entry point that scripts/gen-mcp-configs.sh documents
//      actually exists (it did not — the header told contributors to run a
//      target no Makefile defined).

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFileSync, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..');

// opencode.jsonc is JSON with a leading "// GENERATED ..." banner.
function readJsonc(file) {
  const raw = fs.readFileSync(path.join(root, file), 'utf8');
  return JSON.parse(raw.replace(/^\s*\/\/.*$/gm, ''));
}

test('opencode.jsonc mirrors the canonical .mcp.json server list', () => {
  const canonical = JSON.parse(fs.readFileSync(path.join(root, '.mcp.json'), 'utf8'));
  const adapter = readJsonc('opencode.jsonc');

  const canonicalServers = Object.keys(canonical.mcpServers).sort();
  const adapterServers = Object.keys(adapter.mcp.servers).sort();
  assert.deepEqual(adapterServers, canonicalServers);
});

test('generator --check reports the adapters are up to date', () => {
  const out = execFileSync('bash', ['scripts/gen-mcp-configs.sh', '--check'], {
    cwd: root,
    encoding: 'utf8',
  });
  assert.match(out, /All adapters are up-to-date/);
});

test('make mcp resolves to the generator', () => {
  const result = spawnSync('make', ['-n', 'mcp'], { cwd: root, encoding: 'utf8' });
  assert.equal(result.status, 0, `${result.stdout}${result.stderr}`);
  assert.match(result.stdout, /scripts\/gen-mcp-configs\.sh/);
});
