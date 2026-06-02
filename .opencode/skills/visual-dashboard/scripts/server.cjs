'use strict';

const http = require('http');
const fs = require('fs');
const path = require('path');
const { OPCODES, computeAcceptKey, encodeFrame, decodeFrame } = require('./ws-protocol.cjs');
const { createHttpHandler } = require('./http-handler.cjs');

const PORT = process.env.BIGPOWERS_DASHBOARD_PORT || (49152 + Math.floor(Math.random() * 16383));
const HOST = process.env.BIGPOWERS_DASHBOARD_HOST || '127.0.0.1';
const URL_HOST = process.env.BIGPOWERS_DASHBOARD_URL_HOST || (HOST === '127.0.0.1' ? 'localhost' : HOST);
const SESSION_DIR = process.env.BIGPOWERS_DASHBOARD_DIR || '/tmp/bigpowers-dashboard';
const CONTENT_DIR = path.join(SESSION_DIR, 'content');
const STATE_DIR = path.join(SESSION_DIR, 'state');
const PROJECT_DIR = process.env.BIGPOWERS_PROJECT_DIR
  ? path.resolve(process.env.BIGPOWERS_PROJECT_DIR)
  : null;
let ownerPid = process.env.BIGPOWERS_DASHBOARD_OWNER_PID
  ? Number(process.env.BIGPOWERS_DASHBOARD_OWNER_PID)
  : null;

const frameTemplate = fs.readFileSync(path.join(__dirname, 'frame-template.html'), 'utf-8');
const helperScript = fs.readFileSync(path.join(__dirname, 'helper.js'), 'utf-8');
const helperInjection = '<script>\n' + helperScript + '\n</script>';
const cockpitTemplate = fs.readFileSync(path.join(__dirname, 'cockpit.html'), 'utf-8');

const clients = new Set();
const IDLE_TIMEOUT_MS = 30 * 60 * 1000;
const debounceTimers = new Map();
let lastActivity = Date.now();

function touchActivity() {
  lastActivity = Date.now();
}

const handleRequest = createHttpHandler({
  contentDir: CONTENT_DIR,
  projectDir: PROJECT_DIR,
  cockpitTemplate,
  frameTemplate,
  helperInjection,
  onActivity: touchActivity,
});

function handleMessage(text) {
  let event;
  try {
    event = JSON.parse(text);
  } catch (e) {
    console.error('Failed to parse WebSocket message:', e.message);
    return;
  }
  touchActivity();
  console.log(JSON.stringify({ source: 'user-event', ...event }));
  if (event.choice) {
    const eventsFile = path.join(STATE_DIR, 'events');
    fs.appendFileSync(eventsFile, JSON.stringify(event) + '\n');
  }
}

function broadcast(msg) {
  const frame = encodeFrame(OPCODES.TEXT, Buffer.from(JSON.stringify(msg)));
  for (const socket of clients) {
    try {
      socket.write(frame);
    } catch (e) {
      clients.delete(socket);
    }
  }
}

function handleUpgrade(req, socket) {
  const key = req.headers['sec-websocket-key'];
  if (!key) {
    socket.destroy();
    return;
  }

  const accept = computeAcceptKey(key);
  socket.write(
    'HTTP/1.1 101 Switching Protocols\r\n' +
      'Upgrade: websocket\r\n' +
      'Connection: Upgrade\r\n' +
      'Sec-WebSocket-Accept: ' +
      accept +
      '\r\n\r\n'
  );

  let buffer = Buffer.alloc(0);
  clients.add(socket);

  socket.on('data', (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    while (buffer.length > 0) {
      let result;
      try {
        result = decodeFrame(buffer);
      } catch (e) {
        socket.end(encodeFrame(OPCODES.CLOSE, Buffer.alloc(0)));
        clients.delete(socket);
        return;
      }
      if (!result) break;
      buffer = buffer.slice(result.bytesConsumed);

      switch (result.opcode) {
        case OPCODES.TEXT:
          handleMessage(result.payload.toString());
          break;
        case OPCODES.CLOSE:
          socket.end(encodeFrame(OPCODES.CLOSE, Buffer.alloc(0)));
          clients.delete(socket);
          return;
        case OPCODES.PING:
          socket.write(encodeFrame(OPCODES.PONG, result.payload));
          break;
        case OPCODES.PONG:
          break;
        default: {
          const closeBuf = Buffer.alloc(2);
          closeBuf.writeUInt16BE(1003);
          socket.end(encodeFrame(OPCODES.CLOSE, closeBuf));
          clients.delete(socket);
          return;
        }
      }
    }
  });

  socket.on('close', () => clients.delete(socket));
  socket.on('error', () => clients.delete(socket));
}

function ownerAlive() {
  if (!ownerPid) return true;
  try {
    process.kill(ownerPid, 0);
    return true;
  } catch (e) {
    return e.code === 'EPERM';
  }
}

function writeServerStarted() {
  const baseUrl = 'http://' + URL_HOST + ':' + PORT;
  const cockpitUrl = PROJECT_DIR
    ? baseUrl + '/cockpit.html?projectDir=' + encodeURIComponent(PROJECT_DIR)
    : null;
  const info = JSON.stringify({
    type: 'server-started',
    port: Number(PORT),
    host: HOST,
    url_host: URL_HOST,
    url: baseUrl,
    cockpit_url: cockpitUrl,
    project_dir: PROJECT_DIR,
    screen_dir: CONTENT_DIR,
    state_dir: STATE_DIR,
  });
  console.log(info);
  fs.writeFileSync(path.join(STATE_DIR, 'server-info'), info + '\n');
}

function startServer() {
  if (!fs.existsSync(CONTENT_DIR)) fs.mkdirSync(CONTENT_DIR, { recursive: true });
  if (!fs.existsSync(STATE_DIR)) fs.mkdirSync(STATE_DIR, { recursive: true });

  const knownFiles = new Set(fs.readdirSync(CONTENT_DIR).filter((f) => f.endsWith('.html')));

  const server = http.createServer(handleRequest);
  server.on('upgrade', handleUpgrade);

  const watcher = fs.watch(CONTENT_DIR, (eventType, filename) => {
    if (!filename || !filename.endsWith('.html')) return;

    if (debounceTimers.has(filename)) clearTimeout(debounceTimers.get(filename));
    debounceTimers.set(
      filename,
      setTimeout(() => {
        debounceTimers.delete(filename);
        const filePath = path.join(CONTENT_DIR, filename);
        if (!fs.existsSync(filePath)) return;
        touchActivity();

        if (!knownFiles.has(filename)) {
          knownFiles.add(filename);
          const eventsFile = path.join(STATE_DIR, 'events');
          if (fs.existsSync(eventsFile)) fs.unlinkSync(eventsFile);
          console.log(JSON.stringify({ type: 'screen-added', file: filePath }));
        } else {
          console.log(JSON.stringify({ type: 'screen-updated', file: filePath }));
        }

        broadcast({ type: 'reload' });
      }, 100)
    );
  });
  watcher.on('error', (err) => console.error('fs.watch error:', err.message));

  function shutdown(reason) {
    console.log(JSON.stringify({ type: 'server-stopped', reason }));
    const infoFile = path.join(STATE_DIR, 'server-info');
    if (fs.existsSync(infoFile)) fs.unlinkSync(infoFile);
    fs.writeFileSync(
      path.join(STATE_DIR, 'server-stopped'),
      JSON.stringify({ reason, timestamp: Date.now() }) + '\n'
    );
    watcher.close();
    clearInterval(lifecycleCheck);
    server.close(() => process.exit(0));
  }

  const lifecycleCheck = setInterval(() => {
    if (!ownerAlive()) shutdown('owner process exited');
    else if (Date.now() - lastActivity > IDLE_TIMEOUT_MS) shutdown('idle timeout');
  }, 60 * 1000);
  lifecycleCheck.unref();

  if (ownerPid) {
    try {
      process.kill(ownerPid, 0);
    } catch (e) {
      if (e.code !== 'EPERM') {
        console.log(
          JSON.stringify({
            type: 'owner-pid-invalid',
            pid: ownerPid,
            reason: 'dead at startup',
          })
        );
        ownerPid = null;
      }
    }
  }

  server.listen(PORT, HOST, writeServerStarted);
}

if (require.main === module) {
  startServer();
}

module.exports = { startServer };
