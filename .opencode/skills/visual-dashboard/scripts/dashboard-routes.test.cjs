'use strict';

const { test, before, after } = require('node:test');
const assert = require('node:assert/strict');
const http = require('http');
const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const projectDir = path.resolve(__dirname, '../../../..');
const scriptsDir = __dirname;

let proc;
let port;

function request(method, urlPath) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      { hostname: '127.0.0.1', port, path: urlPath, method },
      (res) => {
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () =>
          resolve({
            status: res.statusCode,
            headers: res.headers,
            body: Buffer.concat(chunks).toString('utf8'),
          })
        );
      }
    );
    req.on('error', reject);
    req.end();
  });
}

before(async () => {
  const sessionDir = fs.mkdtempSync(path.join(require('os').tmpdir(), 'bp-dash-'));
  const contentDir = path.join(sessionDir, 'content');
  const stateDir = path.join(sessionDir, 'state');
  fs.mkdirSync(contentDir, { recursive: true });
  fs.mkdirSync(stateDir, { recursive: true });

  port = 55000 + Math.floor(Math.random() * 1000);
  proc = spawn(
    'node',
    [path.join(scriptsDir, 'server.cjs')],
    {
      env: {
        ...process.env,
        BIGPOWERS_DASHBOARD_PORT: String(port),
        BIGPOWERS_DASHBOARD_DIR: sessionDir,
        BIGPOWERS_PROJECT_DIR: projectDir,
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    }
  );

  await new Promise((resolve, reject) => {
    const t = setTimeout(() => reject(new Error('server start timeout')), 5000);
    proc.stdout.on('data', (buf) => {
      if (buf.toString().includes('server-started')) {
        clearTimeout(t);
        resolve();
      }
    });
    proc.on('error', reject);
  });
});

after(() => {
  if (proc && !proc.killed) proc.kill('SIGTERM');
});

test('GET / redirects to cockpit when project dir set and content empty', async () => {
  const res = await request('GET', '/');
  assert.equal(res.status, 302);
  assert.match(res.headers.location, /\/cockpit\.html\?projectDir=/);
});

test('GET /favicon.ico returns 204', async () => {
  const res = await request('GET', '/favicon.ico');
  assert.equal(res.status, 204);
});
