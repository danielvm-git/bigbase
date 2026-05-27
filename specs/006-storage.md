# Slice 6: Storage — "See Files"

**type:** epic  
**status:** planned  
**verify:** `curl -X POST /api/storage/upload -F "file=@photo.png"` → file URL

## Purpose

File upload, storage, and retrieval. Local filesystem backend with image thumbnail support.

## Scope

- File upload (`POST /api/storage/upload`)
- File download (`GET /api/storage/files/:id`)
- File listing (`GET /api/storage/files`)
- File deletion (`DELETE /api/storage/files/:id`)
- Image thumbnails (auto-generated on upload)
- MIME type validation
- File size limits (configurable, default 10 MB)

## Design Decisions

- **Local filesystem** as primary backend (no S3 dependency)
- Files stored in `data/storage/` directory
- Metadata in SQLite (`storage_files` table)
- Thumbnails generated with Go `image` stdlib
- Files referenceable via UUID

## Implementation Plan

### components/storage/storage.go

```go
type Storage struct {
    db     *db.DB
    logger Logger
    dir    string
    maxSize int64
}

type FileInfo struct {
    ID         string    `json:"id"`
    Name       string    `json:"name"`
    Size       int64     `json:"size"`
    MimeType   string    `json:"mime_type"`
    Thumbnails []string  `json:"thumbnails,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/storage/upload` | Yes | Upload file(s) |
| GET | `/api/storage/files/:id` | Yes | Download file |
| GET | `/api/storage/files` | Yes | List files |
| DELETE | `/api/storage/files/:id` | Yes | Delete file |
| GET | `/api/storage/files/:id/thumb/:size` | Yes | Get thumbnail |

### Auto-migration

```sql
CREATE TABLE IF NOT EXISTS storage_files (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    path TEXT NOT NULL,
    thumbnails TEXT DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

## Verify

```bash
# Upload
curl -X POST /api/storage/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@photo.png"

# Download
curl -H "Authorization: Bearer $TOKEN" \
  -o /tmp/downloaded.png \
  /api/storage/files/<uuid>

# List
curl -H "Authorization: Bearer $TOKEN" /api/storage/files
```
