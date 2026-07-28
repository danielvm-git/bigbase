#!/usr/bin/env python3
"""Tier-1 mechanical certification scorer for the closed-bug sweep.

Joins specs/bugs/registry.yaml against the Tier-0 test evidence and emits a
deterministic verdict per closed bug. No LLM, no judgment calls: running this
twice on identical inputs must produce byte-identical output.

Usage:  python3 specs/verifications/.sweep/score.py
Output: specs/verifications/.sweep/scorecard.json
"""

from __future__ import annotations

import difflib
import json
import os
import re
import subprocess
import sys
from pathlib import Path

import yaml

REPO = Path(__file__).resolve().parents[3]
SWEEP = REPO / "specs/verifications/.sweep"
REGISTRY = REPO / "specs/bugs/registry.yaml"
PINNED = "cedb0fc51cca0436c709295c79dd251906536513"

FUZZY_FLOOR = 0.6

# Package test scripts that are `echo ok` stubs -- zero evidence value.
STUB_PACKAGES = {
    "packages/auth-react",
    "packages/auth-vue",
    "packages/auth-astro",
    "packages/auth-ui-svelte",
    "packages/auth-svelte",
}

# Scopes whose behaviour is DB-driver-sensitive; without the postgres leg they
# cannot reach full CERTIFIED.
DB_SENSITIVE = {"auth", "sites", "deploy", "db"}

CLOSED = {"fixed", "done"}

# --------------------------------------------------------------------------
# Guard classification
# --------------------------------------------------------------------------

KIND_GO_TEST_FUNC = "GO_TEST_FUNC"
KIND_BARE_TEST_FILE = "BARE_TEST_FILE"
KIND_NONTEST_GO_FILE = "NONTEST_GO_FILE"
KIND_TS_TEST = "TS_TEST"
KIND_SHELL_CMD = "SHELL_CMD"
KIND_NONTEST_ARTIFACT = "NONTEST_ARTIFACT"
KIND_DICT_MANUAL = "DICT_MANUAL"
KIND_PROSE = "PROSE"
KIND_MALFORMED = "MALFORMED"

RE_GO_FUNC = re.compile(r"^(\S+_test\.go)\s+(\S+)\s*$")
RE_GO_BARE = re.compile(r"^(\S+_test\.go)\s*$")
RE_GO_NONTEST = re.compile(r"^(\S+\.go)\s+(Test\S*)\s*$")
RE_TS = re.compile(r"^(\S+\.(?:test|spec|contract\.test)\.(?:ts|tsx|js))\b(.*)$")
RE_SHELL = re.compile(r"^\s*(go\s+(test|build|vet)|npm|npx|bun|curl|bash|\./|make|golangci-lint|gosec)\b")
RE_ARTIFACT = re.compile(r"^(\S+\.(?:ya?ml|md|json|toml))\s*$")
RE_TESTFUNC_DECL = re.compile(r"^func\s+(Test\w+)\s*\(", re.M)


def classify(guard) -> str:
    if isinstance(guard, dict):
        return KIND_DICT_MANUAL
    g = str(guard).strip()
    if not g:
        return KIND_MALFORMED
    if RE_GO_FUNC.match(g):
        return KIND_GO_TEST_FUNC
    if RE_GO_BARE.match(g):
        return KIND_BARE_TEST_FILE
    if RE_GO_NONTEST.match(g):
        return KIND_NONTEST_GO_FILE
    if RE_TS.match(g):
        return KIND_TS_TEST
    if RE_SHELL.match(g):
        return KIND_SHELL_CMD
    if RE_ARTIFACT.match(g):
        return KIND_NONTEST_ARTIFACT
    # Anything left that names no path and no Test token is prose.
    if "/" not in g and "Test" not in g:
        return KIND_PROSE
    return KIND_MALFORMED


# --------------------------------------------------------------------------
# Tier-0 evidence indices
# --------------------------------------------------------------------------


def go_module_path() -> str:
    for line in (REPO / "go.mod").read_text().splitlines():
        if line.startswith("module "):
            return line.split(None, 1)[1].strip()
    raise RuntimeError("module path not found in go.mod")


def load_go_results(path: Path):
    """(package, test) -> outcome, from `go test -json` events.

    A test's final outcome is the LAST terminal action seen for it. Skips are
    tracked as their own state and are never coerced to PASS.
    """
    results: dict[tuple[str, str], str] = {}
    pkg_results: dict[str, str] = {}
    if not path.exists():
        return results, pkg_results
    with path.open() as fh:
        for line in fh:
            line = line.strip()
            if not line or not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                continue
            action = ev.get("Action")
            if action not in ("pass", "fail", "skip"):
                continue
            pkg = ev.get("Package", "")
            test = ev.get("Test")
            if test:
                results[(pkg, test)] = action.upper()
            else:
                pkg_results[pkg] = action.upper()
    return results, pkg_results


def load_vitest(paths: list[Path]) -> dict[str, str]:
    """fullName -> outcome for every vitest assertion result."""
    out: dict[str, str] = {}
    for p in paths:
        if not p.exists():
            continue
        try:
            data = json.loads(p.read_text())
        except json.JSONDecodeError:
            continue
        for suite in data.get("testResults", []):
            fname = suite.get("name", "")
            for a in suite.get("assertionResults", []):
                status = a.get("status", "")
                outcome = {"passed": "PASS", "failed": "FAIL"}.get(status, "SKIP")
                full = a.get("fullName") or a.get("title") or ""
                out[f"{fname}::{full}"] = outcome
                out.setdefault(full, outcome)
    return out


def tests_defined_in_file(rel_path: str) -> list[str]:
    f = REPO / rel_path
    if not f.exists():
        return []
    try:
        return RE_TESTFUNC_DECL.findall(f.read_text(errors="replace"))
    except OSError:
        return []


def normalize_tokens(name: str) -> set[str]:
    spaced = re.sub(r"(?<!^)(?=[A-Z])", " ", name).replace("_", " ")
    return {t for t in spaced.lower().split() if t and t != "test"}


def fuzzy_match(target: str, candidates: list[str]):
    """Best candidate by token Jaccard + sequence ratio. Returns (name, score)."""
    best, best_score = None, 0.0
    tt = normalize_tokens(target)
    for c in candidates:
        ct = normalize_tokens(c)
        jac = len(tt & ct) / len(tt | ct) if (tt | ct) else 0.0
        seq = difflib.SequenceMatcher(None, target.lower(), c.lower()).ratio()
        score = max(jac, seq * 0.9)
        if score > best_score:
            best, best_score = c, score
    return best, round(best_score, 3)


# --------------------------------------------------------------------------
# Guard resolution
# --------------------------------------------------------------------------


def pkg_for_path(rel_path: str, module: str) -> str:
    d = os.path.dirname(rel_path)
    return f"{module}/{d}" if d else module


def aggregate(outcomes: list[str]) -> str:
    if not outcomes:
        return "UNRESOLVED"
    if "FAIL" in outcomes:
        return "FAIL"
    if "SKIP" in outcomes:
        return "SKIP"
    return "PASS"


def in_stub_package(rel_path: str) -> bool:
    return any(rel_path.startswith(s + "/") for s in STUB_PACKAGES)


def resolve_guard(guard, go_tests, go_pkgs, vitest, module) -> dict:
    kind = classify(guard)
    row = {
        "guard_raw": guard if isinstance(guard, str) else json.dumps(guard, sort_keys=True),
        "resolved_kind": kind,
        "resolved_test_id": None,
        "exists": False,
        "outcome": "UNRESOLVED",
        "confidence": 0.0,
        "fuzzy_candidate": None,
        "fuzzy_score": None,
        "note": "",
    }
    g = row["guard_raw"].strip()

    if kind == KIND_DICT_MANUAL:
        row["outcome"] = "NOT_AUTOMATABLE"
        row["note"] = "manual instruction; requires Tier-2"
        return row

    if kind == KIND_PROSE:
        row["note"] = "free text, no locatable test; requires Tier-2"
        return row

    if kind == KIND_MALFORMED:
        row["note"] = "registry data bug: unparseable guard"
        return row

    if kind == KIND_NONTEST_ARTIFACT:
        row["outcome"] = "EXISTS" if (REPO / g).exists() else "MISSING"
        row["confidence"] = 0.3
        row["note"] = "not a test; cannot certify behaviour"
        return row

    if kind == KIND_TS_TEST:
        m = RE_TS.match(g)
        path, rest = m.group(1), m.group(2).strip()
        if in_stub_package(path):
            row["resolved_kind"] = "STUB_NOOP"
            row["note"] = "resolves into an `echo ok` stub package; zero evidence"
            return row
        row["exists"] = (REPO / path).exists()
        matches = [k for k in vitest if path.split("/")[-1] in k or path in k]
        if rest:
            narrowed = [k for k in matches if rest.lower() in k.lower()]
            matches = narrowed or matches
        if matches:
            row["resolved_test_id"] = path + (f" :: {rest}" if rest else "")
            row["outcome"] = aggregate([vitest[k] for k in matches])
            row["confidence"] = 0.9 if rest else 0.75
        else:
            row["note"] = "no vitest result matched this file"
        return row

    if kind == KIND_SHELL_CMD:
        targets = re.findall(r"\./[\w./-]+", g)
        pkgs: list[str] = []
        for t in targets:
            t = t.rstrip("/")
            if t in ("./...", "."):
                pkgs = list(go_pkgs)
                break
            rel = t[2:]
            pkgs.append(f"{module}/{rel}")
        outcomes = []
        for p in pkgs:
            outcomes += [v for (pk, _), v in go_tests.items() if pk == p]
            if p in go_pkgs:
                outcomes.append(go_pkgs[p])
        row["resolved_kind"] = KIND_SHELL_CMD
        row["resolved_test_id"] = ",".join(sorted(set(pkgs))) or None
        row["outcome"] = aggregate(outcomes)
        row["confidence"] = 0.7 if outcomes else 0.0
        row["exists"] = bool(outcomes)
        row["note"] = "shell command replayed as package-level aggregate"
        return row

    if kind == KIND_BARE_TEST_FILE:
        # No function named -- aggregate over the tests declared in that file.
        if in_stub_package(g):
            row["resolved_kind"] = "STUB_NOOP"
            row["note"] = "resolves into a stub package; zero evidence"
            return row
        row["exists"] = (REPO / g).exists()
        if not row["exists"]:
            row["note"] = f"test file {g} does not exist in the working tree"
            return row
        pkg = pkg_for_path(g, module)
        funcs = tests_defined_in_file(g)
        outcomes = [go_tests[(pkg, f)] for f in funcs if (pkg, f) in go_tests]
        row["resolved_test_id"] = f"{pkg} [{len(outcomes)}/{len(funcs)} tests]"
        row["outcome"] = aggregate(outcomes)
        row["confidence"] = 0.8 if outcomes else 0.0
        row["note"] = ("file-level guard: names no specific test, "
                       f"aggregated {len(outcomes)} test(s) declared in {g}")
        return row

    # Go test kinds
    if kind == KIND_GO_TEST_FUNC:
        m = RE_GO_FUNC.match(g)
        path, func = m.group(1), m.group(2)
    else:  # KIND_NONTEST_GO_FILE
        m = RE_GO_NONTEST.match(g)
        path, func = m.group(1), m.group(2)
        row["note"] = "guard names a non-test .go file; scanned package instead"
        path = path.replace(".go", "_test.go")

    if in_stub_package(path):
        row["resolved_kind"] = "STUB_NOOP"
        row["note"] = "resolves into a stub package; zero evidence"
        return row

    pkg = pkg_for_path(path, module)
    base_func = func.split("/")[0]

    if (pkg, func) in go_tests:
        row.update(exists=True, resolved_test_id=f"{pkg}.{func}",
                   outcome=go_tests[(pkg, func)], confidence=1.0)
        return row
    if (pkg, base_func) in go_tests and "/" in func:
        # Subtest name given but only the parent ran/reported.
        row.update(exists=True, resolved_test_id=f"{pkg}.{base_func}",
                   outcome=go_tests[(pkg, base_func)], confidence=0.85,
                   note="matched parent test; subtest not separately reported")
        return row

    # Exact miss -> fuzzy, restricted to tests declared in the named file, then
    # falling back to the whole package.
    candidates = tests_defined_in_file(path)
    scope_note = "same file"
    if not candidates:
        pkg_dir = os.path.dirname(path)
        candidates = []
        d = REPO / pkg_dir
        if d.is_dir():
            for f in sorted(d.glob("*_test.go")):
                candidates += tests_defined_in_file(str(f.relative_to(REPO)))
        scope_note = "same package"

    cand, score = fuzzy_match(base_func, candidates)
    row["fuzzy_candidate"] = cand
    row["fuzzy_score"] = score
    if cand and score >= FUZZY_FLOOR:
        row.update(exists=False, resolved_test_id=f"{pkg}.{cand}",
                   outcome=go_tests.get((pkg, cand), "UNRESOLVED"),
                   confidence=score,
                   note=f"NAME DRIFT: registry says '{func}', matched '{cand}' ({scope_note})")
    else:
        row["note"] = (f"func '{func}' not found in {path}; "
                       f"best candidate {cand!r} scored {score} < {FUZZY_FLOOR}")
    return row


# --------------------------------------------------------------------------
# Per-bug inputs
# --------------------------------------------------------------------------

RE_FRONTMATTER = re.compile(r"^---\n(.*?)\n---", re.S)
RE_SHA = re.compile(r"\b([0-9a-f]{7,40})\b")

_git_cache: dict[str, bool] = {}


def path_in_head(rel_path: str) -> bool:
    clean = re.sub(r"\s*\(.*?\)\s*$", "", str(rel_path)).strip()
    if not clean or clean.startswith("/"):
        return False  # absolute paths are not repo files (e.g. /etc/caddy/Caddyfile)
    if clean in _git_cache:
        return _git_cache[clean]
    r = subprocess.run(["git", "cat-file", "-e", f"{PINNED}:{clean}"],
                       cwd=REPO, capture_output=True)
    _git_cache[clean] = r.returncode == 0
    return _git_cache[clean]


def is_ancestor(sha: str) -> bool:
    r = subprocess.run(["git", "merge-base", "--is-ancestor", sha, PINNED],
                       cwd=REPO, capture_output=True)
    return r.returncode == 0


def sha_exists(sha: str) -> bool:
    r = subprocess.run(["git", "cat-file", "-t", sha], cwd=REPO, capture_output=True)
    return r.returncode == 0 and r.stdout.strip() == b"commit"


def md_facts(rel_file: str | None) -> dict:
    out = {"md_exists": False, "md_status": None, "has_resolution": False}
    if not rel_file:
        return out
    p = REPO / rel_file
    if not p.exists():
        return out
    out["md_exists"] = True
    txt = p.read_text(errors="replace")
    out["has_resolution"] = bool(re.search(r"^##+\s*Resolution", txt, re.M | re.I))
    m = RE_FRONTMATTER.match(txt)
    if m:
        try:
            fm = yaml.safe_load(m.group(1)) or {}
            if isinstance(fm, dict):
                out["md_status"] = fm.get("status")
        except yaml.YAMLError:
            pass
    return out


STATUS_EQUIV = {"fixed": {"fixed", "resolved", "done", "closed", "verified"},
                "done": {"done", "fixed", "resolved", "closed", "verified"}}


def status_conflict(reg_status: str, md_status) -> bool:
    if not md_status:
        return False
    return str(md_status).lower() not in STATUS_EQUIV.get(reg_status, {reg_status})


# --------------------------------------------------------------------------
# Verdict rubric -- deterministic, first match wins
# --------------------------------------------------------------------------


def compute_verdict(b: dict, guards: list[dict], facts: dict, postgres_ran: bool) -> tuple[str, str]:
    resolved = [g for g in guards if g["outcome"] in ("PASS", "FAIL", "SKIP")]
    guard_count = len(guards)
    g_exists = any(g["exists"] for g in guards)
    g_any_fail = any(g["outcome"] == "FAIL" for g in resolved)
    g_any_skip = any(g["outcome"] == "SKIP" for g in resolved)
    g_all_pass = bool(resolved) and all(g["outcome"] == "PASS" for g in resolved)
    g_fuzzy = any(g["fuzzy_score"] and g["fuzzy_score"] >= FUZZY_FLOOR and not g["exists"]
                  for g in guards)
    files = [f for f in (b.get("files_changed") or [])]
    f_present = bool(files) and all(path_in_head(f) for f in files)

    # 1 -- md/registry status conflict
    if facts["md_exists"] and status_conflict(b.get("status"), facts["md_status"]):
        return "SUSPECT", f"bug file says status={facts['md_status']!r}, registry says {b.get('status')!r}"

    # 2 -- referenced commit not reachable from the pinned baseline
    for sha in facts["orphan_shas"]:
        return "SUSPECT", f"referenced commit {sha} exists but is not an ancestor of the baseline"

    # 3 -- a guard actually failed
    if g_any_fail:
        failed = [g["resolved_test_id"] or g["guard_raw"] for g in resolved if g["outcome"] == "FAIL"]
        return "REGRESSED", f"guard failed: {', '.join(failed)}"

    # 4/5 -- guards pass
    if resolved and g_all_pass:
        if g_fuzzy:
            return "CERTIFIED-WEAK", "guards pass but only via fuzzy name-drift match"
        if not g_exists:
            return "CERTIFIED-WEAK", "guards pass but none resolved to a named test"
        if not files:
            return "CERTIFIED-WEAK", "guards pass; no files_changed recorded to cross-check"
        if not f_present:
            return "CERTIFIED-WEAK", "guards pass but files_changed paths are absent from the baseline"
        if b.get("scope", "").split(",")[0].strip() in DB_SENSITIVE and not postgres_ran:
            return "CERTIFIED-WEAK", "guards pass on sqlite leg only; postgres leg not run"
        return "CERTIFIED", "named guard(s) exist and pass; files_changed present in baseline"

    # 6 -- a guard was skipped
    if g_any_skip:
        return "UNPROVEN", "guard was SKIPPED, not executed"

    # 7 -- guards listed but nothing resolved
    if guard_count and not resolved:
        return "UNPROVEN", "guards listed but none resolved to an executable test"

    # 8/9 -- no guards at all
    if guard_count == 0:
        if f_present and facts["has_resolution"]:
            return "CERTIFIED-WEAK", "no executable guard; circumstantial (files present + documented resolution)"
        return "UNPROVEN", "no regression guard and insufficient circumstantial evidence"

    return "UNPROVEN", "insufficient evidence"


# --------------------------------------------------------------------------


def main() -> int:
    module = go_module_path()
    go_tests, go_pkgs = load_go_results(SWEEP / "go-sqlite.jsonl")
    postgres_ran = (SWEEP / "go-postgres.jsonl").exists()
    # Node 22 run is authoritative: the Node 26 run produces 6 spurious
    # localStorage failures that are an environment artifact, not regressions.
    vitest = load_vitest([SWEEP / "ui-vitest-node22.json"])

    reg = yaml.safe_load(REGISTRY.read_text())
    bugs = reg["bugs"]

    rows = []
    for b in bugs:
        key = (b.get("id"), b.get("file"))
        status = b.get("status")
        facts = md_facts(b.get("file"))

        # Collect any commit SHAs the entry references, and flag unreachable ones.
        orphan = []
        blob = " ".join(str(b.get(k, "")) for k in ("commit", "commits", "note", "notes"))
        for sha in RE_SHA.findall(blob):
            if len(sha) >= 7 and sha_exists(sha) and not is_ancestor(sha):
                orphan.append(sha)
        facts["orphan_shas"] = sorted(set(orphan))

        guards_raw = b.get("regression_guards") or []
        if isinstance(guards_raw, (str, dict)):
            guards_raw = [guards_raw]
        guards = [resolve_guard(g, go_tests, go_pkgs, vitest, module) for g in guards_raw]

        if status in CLOSED:
            verdict, reason = compute_verdict(b, guards, facts, postgres_ran)
        else:
            verdict, reason = "OUT-OF-SCOPE", f"status={status}"

        rows.append({
            "bug_id": key[0],
            "bug_file": key[1],
            "md_exists": facts["md_exists"],
            "md_status": facts["md_status"],
            "has_resolution": facts["has_resolution"],
            "status": status,
            "severity": b.get("severity"),
            "scope": b.get("scope"),
            "summary": b.get("summary"),
            "files_changed": b.get("files_changed") or [],
            "files_in_baseline": [f for f in (b.get("files_changed") or []) if path_in_head(f)],
            "orphan_shas": facts["orphan_shas"],
            "guards": guards,
            "verdict": verdict,
            "reason": reason,
            "leg_coverage": "sqlite+postgres" if postgres_ran else "sqlite_only",
        })

    # Integrity assertions -- fail loud rather than silently dropping rows.
    assert len(rows) == len(bugs), f"row count {len(rows)} != registry {len(bugs)}"
    closed_rows = [r for r in rows if r["status"] in CLOSED]

    out = {
        "pinned_commit": PINNED,
        "registry_entries": len(bugs),
        "closed_rows": len(closed_rows),
        "postgres_leg_run": postgres_ran,
        "go_tests_indexed": len(go_tests),
        "vitest_indexed": len(vitest),
        "rows": rows,
    }
    (SWEEP / "scorecard.json").write_text(json.dumps(out, indent=2, sort_keys=True) + "\n")

    from collections import Counter
    dist = Counter(r["verdict"] for r in closed_rows)
    print(f"pinned={PINNED[:9]} entries={len(bugs)} closed={len(closed_rows)} "
          f"go_tests={len(go_tests)} vitest={len(vitest)} postgres_leg={postgres_ran}")
    for v in ("REGRESSED", "SUSPECT", "UNPROVEN", "CERTIFIED-WEAK", "CERTIFIED"):
        print(f"  {v:16s} {dist.get(v, 0)}")
    print(f"  {'OUT-OF-SCOPE':16s} {sum(1 for r in rows if r['verdict'] == 'OUT-OF-SCOPE')}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
