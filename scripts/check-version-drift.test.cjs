'use strict';

// Story: e90s07 — version claims must match the toolchain declarations.
// Docs were already covered; these tests pin the *configurations*: the CI
// GO_VERSION/NODE_VERSION pins and the toolchain.toml Go floor. A CI pin that
// drifts from go.mod compiles a different toolchain than the one the repo
// declares, and nothing failed before.

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFileSync, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const script = path.join(root, 'scripts', 'check-version-drift.sh');

function runIn(fixture) {
  const result = spawnSync('bash', [script], {
    cwd: fixture,
    encoding: 'utf8',
    env: { ...process.env, BIGBASE_VERSION_ROOT: fixture },
  });
  return { status: result.status, output: `${result.stdout ?? ''}${result.stderr ?? ''}` };
}

function fixture({ go, nvmrc, ciGo, ciNode, toolchainGo }) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'e90s07-'));
  fs.writeFileSync(path.join(dir, 'go.mod'), `module example\n\ngo ${go}\n`);
  fs.writeFileSync(path.join(dir, '.nvmrc'), `${nvmrc}\n`);
  if (ciGo !== undefined || ciNode !== undefined) {
    fs.mkdirSync(path.join(dir, '.github', 'workflows'), { recursive: true });
    fs.writeFileSync(
      path.join(dir, '.github', 'workflows', 'test-build-release.yml'),
      `env:\n  GO_VERSION: '${ciGo}'\n  NODE_VERSION: '${ciNode}'\n`,
    );
  }
  if (toolchainGo !== undefined) {
    fs.writeFileSync(path.join(dir, 'toolchain.toml'), `[tools.required.go]\nmin = "${toolchainGo}"\n`);
  }
  return dir;
}

test('passes when CI pins match go.mod and .nvmrc', () => {
  const dir = fixture({ go: '1.26.3', nvmrc: '24', ciGo: '1.26', ciNode: '24' });
  const result = runIn(dir);
  assert.equal(result.status, 0, result.output);
});

test('fails when the CI Go pin drifts from go.mod', () => {
  const dir = fixture({ go: '1.26.3', nvmrc: '24', ciGo: '1.24', ciNode: '24' });
  const result = runIn(dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /GO_VERSION/);
});

test('fails when the CI Node pin drifts from .nvmrc', () => {
  const dir = fixture({ go: '1.26.3', nvmrc: '24', ciGo: '1.26', ciNode: '20' });
  const result = runIn(dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /NODE_VERSION/);
});

test('fails when the toolchain.toml Go floor drifts from go.mod', () => {
  const dir = fixture({ go: '1.26.3', nvmrc: '24', toolchainGo: '1.22' });
  const result = runIn(dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /toolchain\.toml/);
});

test('skips configuration checks when the files are absent', () => {
  const dir = fixture({ go: '1.26.3', nvmrc: '24' });
  const result = runIn(dir);
  assert.equal(result.status, 0, result.output);
});

test('the live repository has no version drift', () => {
  const out = execFileSync('bash', [script], { cwd: root, encoding: 'utf8' });
  assert.match(out, /All version claims are consistent/);
});
