# e49s03: Add path traversal defense to file downloads
## Story ID: e49s03 | Epic: e49 | BCPs: 1 | Status: planned

## Summary
The file download handler (handleFileDownload) skips `filepath.Clean` and absolute-prefix validation, unlike the image serving handler which already has these defenses. Apply the same pattern to prevent path traversal even if the DB-stored path is corrupted.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: Normal file download works
  Given a file is stored at data/files/abc123/hello.txt
  When GET /api/storage/files/abc123/download is called
  Then the file is served with correct Content-Type

Scenario: Path traversal attempt is blocked
  Given a record has path ../../../etc/passwd in the DB
  When the download handler serves the file
  Then the response is 403 "access denied"
```
