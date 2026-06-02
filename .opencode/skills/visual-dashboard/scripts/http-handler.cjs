'use strict';

const fs = require('fs');
const path = require('path');
const { readSpecsStatus } = require('./read-specs-status.cjs');
const { WAITING_PAGE } = require('./waiting-page.cjs');

const MIME_TYPES = {
  '.html': 'text/html',
  '.css': 'text/css',
  '.js': 'application/javascript',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
};

function parseQuery(url) {
  const i = url.indexOf('?');
  if (i < 0) return {};
  const q = {};
  for (const part of url.slice(i + 1).split('&')) {
    const [k, v] = part.split('=').map(decodeURIComponent);
    if (k) q[k] = v || '';
  }
  return q;
}

function isFullDocument(html) {
  const trimmed = html.trimStart().toLowerCase();
  return trimmed.startsWith('<!doctype') || trimmed.startsWith('<html');
}

function createHttpHandler(deps) {
  const {
    contentDir,
    projectDir,
    cockpitTemplate,
    frameTemplate,
    helperInjection,
    onActivity,
  } = deps;

  function wrapInFrame(content) {
    return frameTemplate.replace('<!-- CONTENT -->', content);
  }

  function getNewestScreen() {
    const files = fs
      .readdirSync(contentDir)
      .filter((f) => f.endsWith('.html'))
      .map((f) => {
        const fp = path.join(contentDir, f);
        return { path: fp, mtime: fs.statSync(fp).mtime.getTime() };
      })
      .sort((a, b) => b.mtime - a.mtime);
    return files.length > 0 ? files[0].path : null;
  }

  function sendStatus(res, projectDirArg) {
    if (!projectDirArg || !fs.existsSync(path.join(projectDirArg, 'specs'))) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'projectDir must point to a repo with specs/' }));
      return;
    }
    try {
      const body = JSON.stringify(readSpecsStatus(projectDirArg));
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(body);
    } catch (e) {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: e.message }));
    }
  }

  function sendRoot(res, method) {
    const screenFile = getNewestScreen();
    const hasSpecs =
      projectDir && fs.existsSync(path.join(projectDir, 'specs'));

    if (!screenFile && hasSpecs) {
      const qs = '?projectDir=' + encodeURIComponent(projectDir);
      res.writeHead(302, { Location: '/cockpit.html' + qs });
      res.end();
      return;
    }

    let html = screenFile
      ? (raw) => (isFullDocument(raw) ? raw : wrapInFrame(raw))(
          fs.readFileSync(screenFile, 'utf-8')
        )
      : WAITING_PAGE;

    if (html.includes('</body>')) {
      html = html.replace('</body>', helperInjection + '\n</body>');
    } else {
      html += helperInjection;
    }

    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(method === 'HEAD' ? undefined : html);
  }

  function sendFile(res, urlPath, method) {
    const fileName = urlPath.slice(7);
    const filePath = path.join(contentDir, path.basename(fileName));
    if (!fs.existsSync(filePath)) {
      res.writeHead(404);
      res.end('Not found');
      return;
    }
    const ext = path.extname(filePath).toLowerCase();
    const contentType = MIME_TYPES[ext] || 'application/octet-stream';
    res.writeHead(200, { 'Content-Type': contentType });
    res.end(method === 'HEAD' ? undefined : fs.readFileSync(filePath));
  }

  return function handleRequest(req, res) {
    onActivity();
    const urlPath = req.url.split('?')[0];
    const isRead = req.method === 'GET' || req.method === 'HEAD';

    if (isRead && urlPath === '/api/status') {
      const q = parseQuery(req.url);
      const dir = q.projectDir ? path.resolve(q.projectDir) : null;
      sendStatus(res, dir);
      return;
    }

    if (isRead && urlPath === '/cockpit.html') {
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end(req.method === 'HEAD' ? undefined : cockpitTemplate);
      return;
    }

    if (isRead && urlPath === '/favicon.ico') {
      res.writeHead(204);
      res.end();
      return;
    }

    if (isRead && urlPath === '/') {
      sendRoot(res, req.method);
      return;
    }

    if (isRead && urlPath.startsWith('/files/')) {
      sendFile(res, urlPath, req.method);
      return;
    }

    res.writeHead(404);
    res.end('Not found');
  };
}

module.exports = { createHttpHandler, parseQuery };
