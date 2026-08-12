# Kernel Responsibilities

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Kernel Responsibilities
- Component discovery and registration
- Dependency resolution
- Lifecycle management: Init → Start → Stop
- Event bus for hook-based communication
- Config merge: defaults + user overrides
