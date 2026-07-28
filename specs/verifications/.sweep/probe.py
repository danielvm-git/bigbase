#!/usr/bin/env python3
"""Tier-3 behavioural probes against a live local BigBase instance.

Proves security fixes behaviourally rather than trusting a passing unit test.
Requires a scratch server already running (see environment.md for the command).

Usage: python3 specs/verifications/.sweep/probe.py [base_url]
Output: specs/verifications/.sweep/probes.json
"""

from __future__ import annotations

import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

BASE = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:18080"
OUT = Path(__file__).resolve().parent / "probes.json"

results: list[dict] = []


def call(method: str, path: str, body=None, token=None, headers=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            raw = r.read().decode(errors="replace")
            return r.status, dict(r.headers), raw
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), e.read().decode(errors="replace")
    except Exception as e:  # connection refused, timeout, ...
        return 0, {}, str(e)


def record(bug_ids, name, expectation, passed, detail):
    results.append({
        "bug_ids": bug_ids if isinstance(bug_ids, list) else [bug_ids],
        "probe": name,
        "expectation": expectation,
        "result": "PASS" if passed else ("FAIL" if passed is False else "INCONCLUSIVE"),
        "detail": detail,
    })
    mark = {True: "PASS", False: "FAIL"}.get(passed, "INCONCLUSIVE")
    print(f"[{mark:12s}] {name}: {detail}")


def jload(raw):
    try:
        return json.loads(raw)
    except Exception:
        return {}


# ---------------------------------------------------------------- setup
def register(email, password):
    st, _, raw = call("POST", "/api/auth/register",
                      {"email": email, "password": password, "name": email.split("@")[0]})
    d = jload(raw)
    tok = d.get("access_token") or d.get("accessToken") or d.get("token")
    if not tok:
        st2, _, raw2 = call("POST", "/api/auth/login", {"email": email, "password": password})
        d2 = jload(raw2)
        tok = d2.get("access_token") or d2.get("accessToken") or d2.get("token")
        return tok, (st, raw, st2, raw2)
    return tok, (st, raw)


def main() -> int:
    st, _, _ = call("GET", "/health")
    if st != 200:
        print(f"server not reachable at {BASE} (health={st})", file=sys.stderr)
        return 2

    tok_a, dbg_a = register("orga-sweep@example.com", "SweepProbePassw0rd!")
    tok_b, dbg_b = register("orgb-sweep@example.com", "SweepProbePassw0rd!")
    print(f"tokenA={'yes' if tok_a else 'NO'} tokenB={'yes' if tok_b else 'NO'}")
    if not tok_a or not tok_b:
        print(f"  debugA={str(dbg_a)[:300]}")
        print(f"  debugB={str(dbg_b)[:300]}")

    # ---------------- BUG-138 / auth middleware: unauthenticated access denied
    st, _, _ = call("GET", "/api/auth/me")
    record("BUG-138", "unauthenticated /api/auth/me is rejected",
           "401/403 without a token", st in (401, 403),
           f"status={st}")

    # ---------------- BUG-136/BUG-135/BUG-2026-07-24T184443: cross-tenant sites
    if tok_a and tok_b:
        st_a, _, raw_a = call("GET", "/api/sites", token=tok_a)
        st_b, _, raw_b = call("GET", "/api/sites", token=tok_b)
        sites_a, sites_b = jload(raw_a), jload(raw_b)

        def ids(d):
            items = d if isinstance(d, list) else (d.get("sites") or d.get("data") or [])
            return {str(i.get("id")) for i in items if isinstance(i, dict)}

        overlap = ids(sites_a) & ids(sites_b)
        record(["BUG-136", "BUG-135", "BUG-2026-07-24T184443"],
               "site listing is org-scoped",
               "org A and org B see disjoint site sets",
               st_a == 200 and st_b == 200 and not overlap,
               f"statusA={st_a} statusB={st_b} overlap={sorted(overlap) or 'none'}")

        # cross-tenant read of a specific site id
        st_x, _, _ = call("GET", "/api/sites/1", token=tok_b)
        record(["BUG-136", "BUG-135"], "cross-tenant site fetch by id",
               "404/403 when org B requests a site it does not own",
               st_x in (403, 404), f"status={st_x}")

    # ---------------- Seeded cross-tenant fixture (strong IDOR proof)
    # An empty-vs-empty site list proves nothing, so seed a real row owned by
    # one org and confirm a different org cannot read or delete it.
    db = Path(sys.argv[2]) if len(sys.argv) > 2 else None
    if db and db.exists():
        import sqlite3
        owner_tok, _ = register("owner-sweep@example.com", "SweepProbePassw0rd!")
        attacker_tok, _ = register("attacker-sweep@example.com", "SweepProbePassw0rd!")
        con = sqlite3.connect(db)
        owner_org = list(con.execute(
            "SELECT default_org_id FROM users WHERE email='owner-sweep@example.com'"))
        owner_org = owner_org[0][0] if owner_org else None
        con.execute(
            "INSERT OR REPLACE INTO sites (id,name,git_repo_id,production_branch,"
            "root_path,github_full_name,created_at,auth_policy,org_id) VALUES "
            "(9101,'owner-secret-site',1,'main','/','owner/secret',datetime('now'),'public',?)",
            (owner_org,))
        con.commit()

        st_own, _, _ = call("GET", "/api/sites/9101", token=owner_tok)
        st_att, _, _ = call("GET", "/api/sites/9101", token=attacker_tok)
        st_del, _, _ = call("DELETE", "/api/sites/9101", token=attacker_tok)
        record(["BUG-136", "BUG-135", "BUG-133"],
               "cross-tenant IDOR on a REAL seeded site",
               "owner reads it (200); a different org gets 403/404 on read AND delete",
               st_own == 200 and st_att in (403, 404) and st_del in (403, 404),
               f"owner_get={st_own} attacker_get={st_att} attacker_delete={st_del} "
               f"(owner_org={owner_org})")

        # Legacy org_id=0 row: visible to every org by design (see sites.go:352).
        con.execute(
            "INSERT OR REPLACE INTO sites (id,name,git_repo_id,production_branch,"
            "root_path,github_full_name,created_at,auth_policy,org_id) VALUES "
            "(9102,'legacy-zero-org-site',1,'main','/','legacy/zero',datetime('now'),'public',0)")
        con.commit()
        _, _, raw_att = call("GET", "/api/sites", token=attacker_tok)
        sees_legacy = "legacy-zero-org-site" in raw_att
        record("BUG-2026-07-24T184443",
               "legacy org_id=0 site visibility",
               "documented behaviour: org_id=0 rows are visible to EVERY org "
               "(sites.go:352 `WHERE s.org_id = ? OR s.org_id = 0`)",
               None,
               f"unrelated org sees legacy row = {sees_legacy}; this is by design but "
               f"depends on the startup reassignment migration having run")
        con.close()

    # ---------------- BUG-130: cici workflow cross-tenant write
    if tok_b:
        st_c, _, _ = call("PUT", "/api/cici/1/workflows",
                          {"name": "sweep", "steps": []}, token=tok_b)
        record("BUG-130", "cici cross-tenant workflow write",
               "403/404 (never 200/201) writing a workflow to another org's repo",
               st_c in (401, 403, 404), f"status={st_c}")

    # ---------------- BUG-2026-07-24-deploy-key-organization-required
    st_d, _, _ = call("GET", "/api/deploy/nonexistent-deployment-id", token=tok_a)
    record("BUG-2026-07-24-deploy-404-site-not-found",
           "unknown deployment id",
           "404, not 500 or a leaked internal error", st_d in (401, 403, 404),
           f"status={st_d}")

    # deploy key with no org context must not be accepted
    st_k, _, _ = call("GET", "/api/sites", token="bb_dep_invalidsweepkey000000")
    record("BUG-2026-07-24-deploy-key-organization-required",
           "invalid bb_dep_ key rejected",
           "401/403 for a forged deploy key", st_k in (401, 403),
           f"status={st_k}")

    # ---------------- BUG-2026-06-12T150000 / CSP + security headers
    st_h, hdrs, _ = call("GET", "/")
    lower = {k.lower(): v for k, v in hdrs.items()}
    csp = lower.get("content-security-policy")
    record(["BUG-2026-06-12T150000", "BUG-2026-07-11T020132"],
           "CSP header present on HTML routes",
           "Content-Security-Policy header is set",
           bool(csp), f"status={st_h} csp={(csp or 'ABSENT')[:110]}")

    xcto = lower.get("x-content-type-options")
    record("BUG-2026-07-11T020149-05", "security headers present",
           "X-Content-Type-Options: nosniff", xcto == "nosniff",
           f"x-content-type-options={xcto!r}")

    # ---------------- BUG-2026-07-10T160003: login brute-force lockout
    codes = []
    for _ in range(12):
        s, _, _ = call("POST", "/api/auth/login",
                       {"email": "orga-sweep@example.com", "password": "wrong-password"})
        codes.append(s)
    locked = any(c == 429 for c in codes) or codes[-1] != codes[0]
    record("BUG-2026-07-10T160003", "login brute-force protection",
           "repeated bad logins are throttled/locked (429 or changed status)",
           locked, f"statuses={codes}")

    # ---------------- BUG-2026-07-10T160005: internal errors not leaked
    st_e, _, raw_e = call("GET", "/api/sites/not-a-number", token=tok_a)
    leak = any(s in raw_e.lower() for s in
               ("goroutine", "panic:", "/users/", "sql:", ".go:", "syntax error"))
    record("BUG-2026-07-10T160005", "internal error details not leaked",
           "no stack trace / SQL / filesystem path in the API error body",
           not leak, f"status={st_e} body={raw_e[:140]!r}")

    # ---------------- static directory listing
    st_s, _, raw_s = call("GET", "/assets/")
    listing = "Index of" in raw_s or "<title>Directory" in raw_s
    record("BUG-2026-07-24-static-directory-listing",
           "no static directory listing",
           "directory paths do not render an index listing",
           not listing, f"status={st_s} listing_markers={listing}")

    OUT.write_text(json.dumps(results, indent=2) + "\n")
    npass = sum(1 for r in results if r["result"] == "PASS")
    nfail = sum(1 for r in results if r["result"] == "FAIL")
    ninc = sum(1 for r in results if r["result"] == "INCONCLUSIVE")
    print(f"\nprobes: {npass} PASS / {nfail} FAIL / {ninc} INCONCLUSIVE  -> {OUT}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
