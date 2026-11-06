# Before Starting a Task

source: AGENTS.md
references: [AGENTS.md]

### Before Starting a Task
| Task Type | First Call (when ctxo is available) |
|---|---|
| Fixing a bug | `get_context_for_task(taskType: "fix")` |
| Adding/extending a feature | `get_context_for_task(taskType: "extend")` |
| Refactoring | `get_context_for_task(taskType: "refactor")` |
| Understanding code | `get_context_for_task(taskType: "understand")` |
