# Any registry — same pattern

source: AGENTS.md
references: [AGENTS.md]

# Any registry — same pattern
rg "dispatch"     $(opensrc path pypi:requests)
cat               $(opensrc path crates:serde)/src/lib.rs
grep -r "Router"  $(opensrc path vercel/next.js)/packages/next/src/
