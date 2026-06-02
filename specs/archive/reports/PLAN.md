---
type: implementation-plan
context: configure bigbase.click domain with HTTPS via Caddy
---

# Domain Setup: bigbase.click

Configure the new `bigbase.click` domain across all layers: DNS → reverse proxy → app config → landing page → docs.

## Steps

| # | Layer | What | Verify |
|---|-------|------|--------|
| 1 | 🌐 DNS (Spaceship) | Create A record `bigbase.click` → `89.116.26.187` | `dig +short bigbase.click` returns `89.116.26.187` |
| 2 | 🔄 Caddy (VPS) | Update Caddyfile: replace `http://` with `bigbase.click` for auto HTTPS via Let's Encrypt | `curl -sI https://bigbase.click | head -1` returns 200 |
| 3 | ⚙️ Config defaults | Update `config/defaults.jsonc` `proxy.domain` from `"localhost"` to `"bigbase.click"`, `proxy.ssl` to `true` | `go run . status` shows domain config |
| 4 | 🏠 Landing page | Replace hardcoded `http://localhost:8080/admin/` with relative `/admin/` in proxy.go docs template | `go test ./components/proxy/...` passes |
| 5 | 📄 DEPLOY.md | Update with new domain and HTTPS URLs | File diff verified |
| 6 | ✅ Commit & deploy | Build, test, commit, push to trigger deploy | CI passes, site live at https://bigbase.click |

## Service file note

The systemd service (`/etc/systemd/system/bigbase.service`) on the VPS already passes `--port 8080` and doesn't need a domain flag — domain is handled entirely by the Caddy reverse proxy. No changes needed there.

## Caddyfile change

**Before:**
```
http:// {
    reverse_proxy 127.0.0.1:8080 { ... }
}
```

**After:**
```
bigbase.click {
    reverse_proxy 127.0.0.1:8080 { ... }
}
```

Caddy will auto-provision a Let's Encrypt TLS certificate for `bigbase.click` and redirect HTTP → HTTPS.
