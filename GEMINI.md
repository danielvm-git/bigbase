# BigBase — Gemini CLI

**THIN ADAPTER — canonical engineering rules live in [AGENTS.md](AGENTS.md).**
The `@AGENTS.md` import below pulls in the full project, tooling (sqz, RTK, bts,
opensrc, ctxo), and workflow rules. Keep only Gemini-specific overrides here.

@AGENTS.md

## Model Routing Matrix (Gemini)

### 1. Model Matrix & Allocation

| Task Category | Optimal Model | Selection Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Gemini 3.1 Pro (High)` | Deep reasoning, complex design trade-offs, and architectural planning. |
| **Codebase Context Search** | `Gemini 3.1 Pro (Low)` | 2M+ context window. Best for reading large file trees and cross-component structures. |
| **Feature Coding & TDD Loops** | `Gemini 3.1 Pro (High)` | Precision coding, exact syntax execution, and deep test writing. |
| **Verification & Utility Tasks** | `Gemini 3.5 Flash (High/Medium)` | Ultra-low latency, cheap tokens. Best for running linters, tests, and compiling. |
| **Browser UI Testing** | `Gemini 3.5 Flash (Low)` | Fast image/visual processing for browser subagents. |
| **Structured Docs & Summaries** | `Gemini 3.5 Flash (Medium)` or `Gemini 3.1 Pro (Low)` | Strong at structured prose, reports, and YAML/JSON synthesis. |

### 2. Dynamic Delegation Protocol

When spawning sub-agents (via `delegate-task`, `dispatch-agents`, or `browser_subagent`):
- **File-system audits, log inspection, linting**: use `Gemini 3.5 Flash (Low)` to keep token costs minimal.
- **Code generation/refactoring sub-tasks**: use `Gemini 3.1 Pro (High)`.
- **Context shaving**: never pass entire files to sub-agents unless they are targets for modification. Pass only specific function signatures or YAML spec blocks.
