# Model Routing Matrix

source: AGENTS.md
references: [AGENTS.md]

## Model Routing Matrix

| Task Category | Optimal Model | Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Claude Opus 4.6 (Thinking)` or `Gemini 3.1 Pro (High)` | Deep reasoning, architecture trade-offs |
| **Codebase Context Search** | `Claude Sonnet 4.6` or `Gemini 3.1 Pro (Low)` | Balanced context window + speed |
| **Feature Coding & TDD Loops** | `Claude Sonnet 4.6 (Thinking)` or `Gemini 3.1 Pro (High)` | Precision coding, exact syntax execution |
| **Verification & Utility Tasks** | `Claude Haiku 3.5` or `Gemini 3.5 Flash` | Low latency for lint/test/compile |
| **Browser UI Testing** | `Claude Haiku 3.5` or `Gemini 3.5 Flash (Low)` | Fast image/visual processing |
