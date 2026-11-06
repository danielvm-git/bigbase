'use strict';

// Story: e90s02 — unresolved merge-conflict markers must fail the gate, and
// legitimate setext/H1 candidates in CHANGELOG.md and archived specs must not
// produce false positives. Pins the reusable gate so CI and pre-commit share
// one implementation instead of two copies of the same regex.

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFileSync, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const script = path.join(root, 'scripts', 'check-conflict-markers.sh');

const run = (args, cwd) => {
  const result = spawnSync('bash', [script, ...args], { cwd, encoding: 'utf8' });
  return { status: result.status, output: `${result.stdout ?? ''}${result.stderr ?? ''}` };
};

const tmpdir = (prefix) => fs.mkdtempSync(path.join(os.tmpdir(), prefix));

const write = (dir, name, content) => {
  const target = path.join(dir, name);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content);
  return target;
};

test('passes on files without conflict markers', () => {
  const dir = tmpdir('e90s02-clean-');
  const file = write(dir, 'clean.go', 'package clean\n');
  const result = run([file], dir);
  assert.equal(result.status, 0, result.output);
});

test('fails on an opening marker', () => {
  const dir = tmpdir('e90s02-open-');
  const file = write(dir, 'bad.go', 'package bad\n<<<<<<< HEAD\nx := 1\n');
  const result = run([file], dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /bad\.go/);
});

test('fails on a bare separator line', () => {
  const dir = tmpdir('e90s02-sep-');
  const file = write(dir, 'bad.yml', 'a: 1\n=======\nb: 2\n');
  assert.equal(run([file], dir).status, 1);
});

test('fails on a closing marker', () => {
  const dir = tmpdir('e90s02-close-');
  const file = write(dir, 'bad.md', '>>>>>>> feature/x\n');
  assert.equal(run([file], dir).status, 1);
});

test('ignores marker-like text that is not at the start of a line', () => {
  const dir = tmpdir('e90s02-inline-');
  const file = write(dir, 'doc.md', 'Use `<<<<<<<` only inside a fenced code sample: x <<<<<<< y\n');
  const result = run([file], dir);
  assert.equal(result.status, 0, result.output);
});

test('fails on a private-key style separator-free marker inside src', () => {
  const dir = tmpdir('e90s02-src-');
  const file = write(dir, 'components/x/main.go', 'package x\n>>>>>>> main\n');
  assert.equal(run([file], dir).status, 1);
});

// Default mode walks git-tracked files. CHANGELOG.md legitimately contains
// setext underlines, and archived specs are historical, so both are exempt.
test('exempts CHANGELOG.md and specs/archive in git mode', () => {
  const dir = tmpdir('e90s02-git-');
  execFileSync('git', ['init', '-q'], { cwd: dir });
  write(dir, 'CHANGELOG.md', 'Release\n=======\n');
  write(dir, 'specs/archive/old-plan.md', 'Plan\n=======\n');
  write(dir, 'main.go', 'package main\n');
  execFileSync('git', ['add', '-A'], { cwd: dir });
  const result = run([], dir);
  assert.equal(result.status, 0, result.output);
});

test('still fails on a marked tracked source file in git mode', () => {
  const dir = tmpdir('e90s02-git-bad-');
  execFileSync('git', ['init', '-q'], { cwd: dir });
  write(dir, 'main.go', 'package main\n<<<<<<< HEAD\n');
  execFileSync('git', ['add', '-A'], { cwd: dir });
  const result = run([], dir);
  assert.equal(result.status, 1, result.output);
  assert.match(result.output, /main\.go/);
});
