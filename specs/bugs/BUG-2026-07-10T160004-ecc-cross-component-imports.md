---
bug_id: BUG-2026-07-10T160004
status: fixed
severity: high
scope: auth,mcp,monitoring,sites
title: Four production cross-component direct imports violate ECC
---

# BUG-2026-07-10T160004: ECC cross-component imports

## Problem

Production code imported sibling components (auth→mcp, mcp→deploy/sites, monitoring→deploy, sites→deploy), violating ECC isolation.

## Fix

- Local DTOs/interfaces in consumers (mcp SiteInfo/DeploymentResult; monitoring Diagnosis/RelatedEvents)
- Composition-root adapters in `adapters.go`
- Injected `ValidateManifest` seam on sites

## Verify

→ verify: `go build . && go test ./components/auth/ ./components/mcp/ ./components/sites/ ./components/monitoring/ -count=1`
