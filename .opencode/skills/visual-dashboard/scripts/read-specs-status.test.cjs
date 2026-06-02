'use strict';

const { test } = require('node:test');
const assert = require('node:assert/strict');
const path = require('path');
const { readSpecsStatus } = require('./read-specs-status.cjs');

const projectDir = path.resolve(__dirname, '../../../..');

test('e17 epic status is done from execution-status.yaml', () => {
  const data = readSpecsStatus(projectDir);
  const e17 = data.epics.find((e) => e.id === 'e17');
  assert.ok(e17, 'e17 should exist in release plan');
  assert.equal(e17.status, 'done');
});

test('planning_status uses flat string values not objects', () => {
  const data = readSpecsStatus(projectDir);
  assert.equal(typeof data.planning_status.scope, 'string');
  assert.equal(data.planning_status.scope, 'done');
});

test('state exposes git and handoff for cockpit header', () => {
  const data = readSpecsStatus(projectDir);
  assert.equal(data.state.git.branch, 'main');
  assert.ok(data.state.git.hash);
  assert.equal(data.state.handoff.next_skill, 'kickoff-branch');
});

test('next_epic_id is first pending when no active epic', () => {
  const data = readSpecsStatus(projectDir);
  if (data.active_epic_id) {
    assert.equal(data.next_epic_id, null);
  } else {
    assert.equal(data.next_epic_id, 'e18');
  }
});
