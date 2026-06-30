# Update Brief — e64: Schema Designer

**Prototype file:** `BigBase Console.html`  
**Project:** BigBase Prototype (`ec1480a1`)  
**Effort:** Small — enhance existing Schema tab in Data Studio  
**Epic:** e64-schema-designer

---

## What to change

The current Data Studio Schema tab already shows column names and types with [Edit] [Delete] buttons. e64 upgrades this to a full schema editor with DDL warnings and an ERD-lite view.

### Data Studio — Schema tab enhancements

#### Current state (already in prototype)
```
[+ Add column]  [Query this]
Name      Type     Actions
id        integer  [Edit] [Delete]
email     text     [Edit] [Delete]
```

#### New state

Add a **View** toggle above the table: `[Columns]  [Diagram]`

**Columns view (default, enhanced):**
```
[Columns]  [Diagram]                              [+ Add column]

Schema preview · collection: users
⚠ Column changes are not persisted until DDL API ships.

Name         Type        Nullable  Default   Actions
id           integer     No        —         [Edit] [Delete]
email        text        No        —         [Edit] [Delete]
created_at   datetime    No        now()     [Edit] [Delete]
```

Columns table gains "Nullable" and "Default" columns. Delete now shows a warning modal:

```
┌────────────────────────────────────────────────┐
│ Drop column?                                 ✕ │
│                                                │
│ Dropping "email" from "users" will delete all  │
│ data in that column. This cannot be undone.    │
│                                                │
│ [Cancel]   [Drop column]  ← danger             │
└────────────────────────────────────────────────┘
```

**Diagram view (new):**
Simple ERD card per collection — no foreign key lines yet (those come later):

```
┌──────────────────────┐  ┌──────────────────────┐
│ users                │  │ posts                │
│ ───────────────────  │  │ ─────────────────── │
│ 🔑 id       integer  │  │ 🔑 id      integer  │
│    email    text     │  │    title   text     │
│    created  datetime │  │    user_id integer  │
│                      │  │    body    text     │
└──────────────────────┘  └──────────────────────┘
```

- Each collection is a card (fixed 200px wide, auto-height)
- Cards arranged in a wrapping grid (`flex-wrap`)
- `🔑` key icon for the `id` column
- Card header: collection name bold + column count muted (e.g. "users · 3 columns")
- No connecting lines in v1

**Add column modal (unchanged from current, just add Nullable + Default fields):**
```
┌──────────────────────────────┐
│ Add column                 ✕ │
│                              │
│ [Column name *           ]   │
│ [Type: text ▾            ]   │
│ ☑ Nullable                   │
│ [Default value           ]   │
│                              │
│ [Cancel]  [Save]             │
└──────────────────────────────┘
```
