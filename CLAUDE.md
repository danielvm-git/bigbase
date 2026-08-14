# BigBase — Claude Code

**THIN ADAPTER — canonical engineering rules live in [AGENTS.md](AGENTS.md).**
The `@AGENTS.md` import below pulls in the full project, tooling (sqz, RTK, bts,
opensrc, ctxo), and workflow rules. Keep only Claude-specific overrides here.

> Note: `sqz init`, `rtk init`, and `ctxo init` may re-insert their own guidance
> blocks into this file. If that happens, the canonical copies in AGENTS.md still
> win — delete the re-added blocks here rather than editing them.

@AGENTS.md

## Model Routing Matrix (Anthropic)

### 1. Model Matrix & Allocation

| Task Category | Optimal Model | Selection Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Claude Opus 4.6 (Thinking)` | Deep reasoning, complex design trade-offs, and architectural planning. |
| **Codebase Context Search** | `Claude Sonnet 4.6 (Thinking)` | Best for reading file trees and cross-component structures. |
| **Feature Coding & TDD Loops** | `Claude Sonnet 4.6 (Thinking)` | Precision coding, exact syntax execution, and deep test writing. |
| **Verification & Utility Tasks** | `Claude Haiku 4.5` | Ultra-low latency, cheap tokens. Best for running linters, tests, and compiling. |
| **Browser UI Testing** | `Claude Sonnet 4.6` | Visual processing for browser subagents. |
| **Structured Docs & Summaries** | `Claude Sonnet 4.6` or `Claude Haiku 4.5` | Strong at structured prose, reports, and YAML/JSON synthesis. |

### 2. Dynamic Delegation Protocol

When spawning sub-agents (via `delegate-task`, `dispatch-agents`, or `browser_subagent`):
- **File-system audits, log inspection, linting**: use `Claude Haiku 4.5` to keep token costs minimal.
- **Code generation/refactoring sub-tasks**: use `Claude Sonnet 4.6 (Thinking)`.
- **Context shaving**: never pass entire files to sub-agents unless they are targets for modification. Pass only specific function signatures or YAML spec blocks.
