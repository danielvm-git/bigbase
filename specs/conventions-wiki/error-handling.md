# Error Handling

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Error Handling
```go
// Always check errors
if err != nil {
    return fmt.Errorf("context: %w", err)
}

// Use sentinel errors for expected failures
var ErrNotFound = errors.New("not found")
```
