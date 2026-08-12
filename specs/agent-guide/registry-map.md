# Registry Map

source: AGENTS.md
references: [AGENTS.md]

### Registry Map

| Registry | Spec | Example |
|----------|------|---------|
| npm | `<name>` or `npm:<name>` | `opensrc path zod` |
| npm (scoped) | `@scope/name` | `opensrc path @trpc/server` |
| PyPI | `pypi:<name>` / `pip:` / `python:` | `opensrc path pypi:requests` |
| crates.io | `crates:<name>` / `cargo:` / `rust:` | `opensrc path crates:serde` |
| GitHub | `owner/repo` or full URL | `opensrc path vercel/next.js` |
| GitLab | `gitlab:owner/repo` or URL | `opensrc path gitlab:gnome/gtk` |
| Bitbucket | `bitbucket:owner/repo` or URL | `opensrc path bitbucket:atlassian/python-bitbucket` |
