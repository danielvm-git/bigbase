# e49s03: Add path traversal defense to file downloads

## 1. Story ID
e49s03

## 2. Epic
e49 — Security: Auth Hardening (Anonymous, OAuth, Path Traversal)

## 3. Status
planned

## 4. BCPs
1

## 5. Type
fix

## 6. Context
domain

## 7. Summary
The file download handler `handleFileDownload` in `components/storage/storage.go` joins the DB-stored file path with the storage directory without `filepath.Clean` or absolute-path boundary validation. This allows path traversal if a corrupted DB record contains `../../../etc/passwd`. The thumbnail handler (`handleThumbnail`) already has the correct defense pattern — apply it to `handleFileDownload` too.

## 8. Problem Statement
In `handleFileDownload` (storage.go):
```go
fullPath := filepath.Join(s.dir, filePath)
```

`filepath.Join` does NOT clean traversal sequences like `..`. If the DB stores `path = "../../../etc/passwd"`, then `fullPath = "data/storage/../../../etc/passwd"` resolves to `/etc/passwd`.

Compare with `handleThumbnail` which already has the correct defense:
```go
fullPath := filepath.Join(s.dir, filepath.Clean(filePath))
absDir, _ := filepath.Abs(s.dir)
absPath, _ := filepath.Abs(fullPath)
if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
    writeJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
    return
}
```

## 9. Proposed Solution
Apply the same three-step defense from `handleThumbnail` to `handleFileDownload`:
1. `filepath.Clean` normalizes `/./`, `/../`, double separators
2. `filepath.Abs` resolves to absolute path
3. Prefix check ensures resolved path stays under storage directory

## 10. Affected Modules
| Module | Purpose | Callers | Contracts |
|--------|---------|---------|-----------|
| `components/storage/` | File upload/download/delete, MIME validation | HTTP clients via `/api/storage/*`; kernel lifecycle | Files stored under `s.dir/<id>/<filename>`; `handleThumbnail` has path traversal defense (the pattern to replicate); `handleFileDownload` serves files directly |

## 11. Dependencies
- **Zero new external packages** — uses `path/filepath` (stdlib) and `strings` (stdlib)

## 12. Implementation Steps

### Story e49s03: Path traversal defense for file downloads — Implementation Steps

**type:** fix
**context:** domain
**Context**: The `handleFileDownload` handler is missing path traversal defense that `handleThumbnail` already has. Apply the same three-step pattern (Clean → Abs → prefix check) to prevent serving files outside the storage directory even if the DB-stored path is corrupted or malicious.

## Steps

1. Add `TestFileDownloadPathTraversal` — table-driven test: normal path works, `../../../etc/passwd` blocked, `./` normalized, absolute path blocked → verify: `go test ./components/storage/ -run TestFileDownloadPathTraversal -v`

2. Apply path traversal defense (Clean → Abs → prefix check) to `handleFileDownload`, matching `handleThumbnail` pattern → verify: `go test ./components/storage/ -run TestFileDownloadPathTraversal -v`

3. Run full storage test suite to confirm no regressions → verify: `go test ./components/storage/ -v`

## Verification Script (Step-by-Step)

1. **Start BigBase**:
   ```bash
   go run . serve --port 9999 &
   sleep 3
   ```
2. **Upload a test file**:
   ```bash
   echo "test content" > /tmp/test.txt
   FILE_ID=$(curl -s -F "file=@/tmp/test.txt" http://localhost:9999/api/storage/upload | jq -r '.id')
   echo "File ID: $FILE_ID"
   ```
3. **Download normally**:
   ```bash
   curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/api/storage/files/$FILE_ID
   # Expected: 200
   ```
4. **Verify path traversal blocked (unit test)**:
   ```bash
   go test ./components/storage/ -run TestFileDownloadPathTraversal -v
   # Expected: PASS — all traversal attempts return 403
   ```
5. **Stop server**:
   ```bash
   kill %1
   ```

## Out of scope
- Path traversal in `handleFileDelete` (uses `filepath.Join(s.dir, id)` where `id` is hex-only — not exploitable)
- Path traversal in `writeFile` (uses `filepath.Base` on filename in `readUpload`)
- Shared helper extraction (premature abstraction for 2 callers)

## Risks
- **os.Stat before validation**: Current code calls `os.Stat` BEFORE Clean/Abs/prefix check. Mitigation: move validation before `os.Stat`.
- **Performance**: `filepath.Abs` does a syscall on relative paths. Negligible per download.

## 13. Definition of Done
- [x] `handleFileDownload` applies `filepath.Clean` on DB-stored path
- [x] `handleFileDownload` applies `filepath.Abs` + prefix check
- [x] Traversal attempt returns 403 "access denied"
- [x] Normal file download still works (200)
- [x] `go test ./components/storage/ -run TestFileDownloadPathTraversal -v` passes
- [x] Full storage test suite passes (`go test ./components/storage/ -v`)

## 14. Acceptance Criteria (Gherkin)
```gherkin
Feature: Path traversal defense for file downloads

  Scenario: Normal file download works
    Given a file is stored at data/storage/abc123/hello.txt
    When GET /api/storage/files/abc123 is called
    Then the response status is 200

  Scenario: Path traversal via ../ is blocked
    Given a DB record has path ../../../etc/passwd
    When the download handler serves the file
    Then the response status is 403

  Scenario: Absolute path is blocked
    Given a DB record has path /etc/passwd
    When the download handler serves the file
    Then the response status is 403

  Scenario: ./ prefix is normalized
    Given a DB record has path ./hello.txt
    When the download handler serves the file
    Then the response status is 200
```

## 15. Test Strategy
- **Unit**: `TestFileDownloadPathTraversal` — table-driven: normal path (200), `../` (403), absolute (403), `./` normalized (200)
- **Regression**: Full `go test ./components/storage/ -v`

## 16. Rollback Plan
- Revert `handleFileDownload` to use `filepath.Join(s.dir, filePath)` without protection
- Delete `TestFileDownloadPathTraversal`
