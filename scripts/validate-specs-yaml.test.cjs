'use strict';

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execSync } = require('child_process');
const path = require('path');

const root = path.resolve(__dirname, '..');

test('validate-specs-yaml.sh exits 0 on current tree', () => {
  const out = execSync('bash scripts/validate-specs-yaml.sh', {
    cwd: root,
    encoding: 'utf8',
  });
  assert.match(out, /validate-specs-yaml: OK/);
});
