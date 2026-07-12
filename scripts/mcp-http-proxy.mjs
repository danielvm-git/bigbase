#!/usr/bin/env node
const url = process.argv[2];
const headers = {};

for (let i = 3; i < process.argv.length; i++) {
  if (process.argv[i] === "--header" && i + 1 < process.argv.length) {
    const m = process.argv[++i].match(/^([A-Za-z0-9_-]+):\s*(.*)$/);
    if (m) headers[m[1]] = m[2];
  }
}

// Log startup to stderr for debugging
process.stderr.write(`mcp-proxy: url=${url}\n`);
process.stderr.write(`mcp-proxy: headers=${JSON.stringify(Object.keys(headers))}\n`);

let buf = "";
process.stdin.on("data", (chunk) => {
  buf += chunk.toString();
  process.stderr.write(`mcp-proxy: got ${chunk.length} bytes\n`);
  const lines = buf.split("\n");
  buf = lines.pop() || "";
  for (const line of lines) {
    if (!line.trim()) continue;
    try { JSON.parse(line); send(line); } catch {}
  }
});

process.stdin.on("end", () => {
  process.stderr.write("mcp-proxy: stdin ended\n");
  if (buf.trim()) { try { JSON.parse(buf); send(buf); } catch {} }
});

async function send(body) {
  process.stderr.write(`mcp-proxy: sending ${body.length} bytes\n`);
  try {
    const res = await fetch(url, {
      method: "POST",
      headers: { ...headers, "Content-Type": "application/json", Accept: "application/json" },
      body,
    });
    const text = await res.text();
    process.stderr.write(`mcp-proxy: got ${text.length} bytes response, status=${res.status}\n`);
    process.stdout.write(text + "\n");
  } catch (err) {
    process.stderr.write(`mcp-proxy: error: ${err.message}\n`);
    const errMsg = JSON.stringify({
      jsonrpc: "2.0",
      id: null,
      error: { code: -32000, message: `Proxy error: ${err.message}` },
    });
    process.stdout.write(errMsg + "\n");
  }
}
