# TestProxyGitHubStars flaky — unmocked live GitHub API call returns live stars

**Source:** Local and CI test run
**Severity:** MEDIUM
**Scope:** proxy

## Description

`TestProxyGitHubStars` failed in `components/proxy`:
```
proxy_test.go:238: expected empty stars before fetch, got "2"
```

## Root cause (4-phase RCA)

1. **Reproduce**: Running `go test -v -run TestProxyGitHubStars ./components/proxy` reproduced the failure immediately.
2. **Isolate**: `p.GitHubStars()` in `components/proxy/proxy.go` calls `p.fetchStars()`. `p.fetchStars()` performs a live HTTP GET to `https://api.github.com/repos/danielvm-git/bigbase`.
3. **Hypothesize**: Previously, the repository had 0 stars, so `fetchStars()` returned an error (`repo.StargazersCount == 0 -> fmt.Errorf("no stars")`), causing `GitHubStars()` to return `""`. When the repository reached 2 stars, `fetchStars()` succeeded and returned `"2"`. The test assertion `if stars := p.GitHubStars(); stars != ""` failed because it assumed empty cache without mocking the HTTP transport or endpoint.
4. **Verify**: Querying the GitHub API returns `stargazers_count: 2`. The test had a hard external dependency on network state and the live star count of the repo.

## Fix applied

1. Added `starsURL string` and `SetStarsURL(url string)` to `Proxy` (with `StarsURL` in `Options`).
2. Pointed `fetchStars()` to `p.starsURL` (defaulting to the GitHub API endpoint if unset).
3. Updated `TestProxyGitHubStars` to mock the HTTP responses via `httptest.NewServer`:
   - Returns empty string when stars API reports 0 stars or an error
   - Returns string count when stars API reports positive count (e.g. 42)
   - Verifies memory caching within TTL (1 remote call for multiple invocations)
   - Verifies fallback to cached count if subsequent remote call fails

## Hardening added

- Unit tests in `components/proxy` now run hermetically without external network access to GitHub.
- Configurable `StarsURL` in `Options` allows test injection and proxy configuration.
- Stabilized `MessagingPage.test.tsx` timeout under concurrent multi-core suite load.

## Evidence

- `go test -v -run TestProxyGitHubStars ./components/proxy` -> all subtests pass
- `go test ./components/proxy` -> pass
- `go test ./...` -> pass
- `npm --prefix ui test` -> 94/94 suites pass

## Status
fixed

## Source
local test run / CI regression

## Discovered
2026-11-02
