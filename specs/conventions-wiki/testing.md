# Testing

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Testing
- Table-driven tests with `t.Run`
- Package name: `package_test`
- Test files: `*_test.go` co-located with source
- For goroutines and supervisors, synchronize with a channel or poll helper before asserting state; never rely on a timing sleep alone.
