'use strict';

// Story: e90s01 — the required-env gate must FAIL when a key declared in
// deploy.yml REQUIRED_ENV_KEYS is missing from .env.example. The happy path is
// already covered by CI; these tests pin the failure path so the gate cannot
// silently degrade into a no-op (the BIGBASE_ROOT_ENCRYPTION_KEY class of
// deploy-time config drift).

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFileSync } = require('child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const script = path.join(root, 'scripts', 'check-required-env-documented.sh');

// runGate executes the gate against a throwaway fixture root instead of the
// real repository, so the tests never depend on the live .env.example.
function runGate(fixture) {
  try {
    const stdout = execFileSync('bash', [script], {
      cwd: fixture,
      encoding: 'utf8',
      env: { ...process.env, BIGBASE_ENV_DOC_ROOT: fixture },
    });
    return { status: 0, output: stdout };
  } catch (err) {
    return { status: err.status, output: `${err.stdout ?? ''}${err.stderr ?? ''}` };
  }
}

function fixture({ deploy, envExample }) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'e90s01-'));
  if (deploy !== undefined) {
    fs.mkdirSync(path.join(dir, '.github', 'workflows'), { recursive: true });
    fs.writeFileSync(path.join(dir, '.github', 'workflows', 'deploy.yml'), deploy);
  }
  if (envExample !== undefined) {
    fs.writeFileSync(path.join(dir, '.env.example'), envExample);
  }
  return dir;
}

const deployWithKeys = (...keys) =>
  [
    'jobs:',
    '  deploy:',
    '    steps:',
    '      - run: |',
    `          REQUIRED_ENV_KEYS=(${keys.join(' ')})`,
    '          for required_key in "${REQUIRED_ENV_KEYS[@]}"; do',
  ].join('\n');

test('passes when every required key is documented', () => {
  const dir = fixture({
    deploy: deployWithKeys('BIGBASE_ROOT_ENCRYPTION_KEY'),
    envExample: 'BIGBASE_ROOT_ENCRYPTION_KEY=\n',
  });
  const result = runGate(dir);
  assert.equal(result.status, 0, result.output);
  assert.match(result.output, /All 1 required env keys documented/);
});

test('fails when a required key is missing from .env.example', () => {
  const dir = fixture({
    deploy: deployWithKeys('BIGBASE_ROOT_ENCRYPTION_KEY', 'BIGBASE_SECOND_KEY'),
    envExample: 'BIGBASE_ROOT_ENCRYPTION_KEY=\n',
  });
  const result = runGate(dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /BIGBASE_SECOND_KEY/);
});

test('fails when a required key is only mentioned without a KEY= entry', () => {
  const dir = fixture({
    deploy: deployWithKeys('BIGBASE_ROOT_ENCRYPTION_KEY'),
    envExample: 'UNRELATED=\n',
  });
  const result = runGate(dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /BIGBASE_ROOT_ENCRYPTION_KEY/);
});

test('accepts a commented placeholder as documentation', () => {
  const dir = fixture({
    deploy: deployWithKeys('BIGBASE_ROOT_ENCRYPTION_KEY'),
    envExample: '# BIGBASE_ROOT_ENCRYPTION_KEY=  (generate with openssl rand -base64 32)\n',
  });
  const result = runGate(dir);
  assert.equal(result.status, 0, result.output);
});

test('skips when the deploy workflow is absent', () => {
  const dir = fixture({ envExample: '' });
  const result = runGate(dir);
  assert.equal(result.status, 0, result.output);
  assert.match(result.output, /SKIP/);
});

test('skips when REQUIRED_ENV_KEYS declares no keys', () => {
  const dir = fixture({ deploy: 'jobs: {}\n', envExample: '' });
  const result = runGate(dir);
  assert.equal(result.status, 0, result.output);
  assert.match(result.output, /SKIP/);
});
