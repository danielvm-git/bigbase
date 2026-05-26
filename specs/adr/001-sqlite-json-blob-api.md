# ADR 001: SQLite + JSON Blob Storage for Auto-CRUD API

type: adr
context: BigBase API component — need to support arbitrary JSON records without predefined schemas (Firebase-like)

## Decision

Use SQLite with a fixed `(id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT)` schema per collection, storing record payloads as JSON text blobs. No schema inference or column-per-field mapping.

## Rationale

- **Zero-config collections**: First write to any collection auto-creates the backing table. No migrations, no DDL in the hot path.
- **Arbitrary JSON**: Clients can store any JSON shape without server-side schema registration.
- **Simplified CRUD**: All four operations collapse to generic SQL over the single `data` column.
- **Pure Go** via `modernc.org/sqlite`: No CGO dependency, single-binary deploy.

## Consequences

- **No SQL-level querying on record fields** — all field-level queries require loading and scanning JSON in Go (acceptable for v0.1; can add SQLite JSON functions later).
- **No type constraints** — fields are unchecked at write time (validation deferred to auth/Slice 4).
- **Table name sanitization** — collection names restricted to `[a-zA-Z0-9_]`; invalid characters rejected with 400.
- **Error handling**: Internal errors are logged server-side; clients receive generic messages per CONVENTIONS.md.
- **PATCH merge semantics**: Record update is a shallow merge — fields in the request body overwrite or add to existing fields. Sending `{"field": null}` sets the field to JSON null; there is no mechanism to delete a field.

## Status

Accepted (Slice 3, commit 696befb).
