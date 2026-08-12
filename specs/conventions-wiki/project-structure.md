# Project Structure

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

## Project Structure
```
bigbase/
├── main.go
├── kernel/
│   ├── kernel.go        — discovery, lifecycle
│   ├── component.go     — Component interface
│   ├── eventbus.go      — hook system
│   ├── config.go        — config merge
│   └── registry.go      — component registration
├── components/
│   ├── proxy/
│   ├── auth/
│   ├── db/
│   ├── api/
│   ├── storage/
│   ├── git/
│   ├── forge/
│   ├── cici/
│   ├── functions/
│   ├── realtime/
│   ├── messaging/
│   ├── deploy/
│   ├── admin/
│   └── monitoring/
├── config/
│   ├── defaults.jsonc
│   └── profiles/
├── specs/               — planning documents
│   ├── adr/             — architecture decision records
│   ├── CONTEXT.md       — domain context
│   ├── RELEASE-PLAN.md  — epics and stories
│   └── ...
└── ui/                  — Admin UI (React)
```
