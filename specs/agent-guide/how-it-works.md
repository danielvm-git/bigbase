# How It Works

source: AGENTS.md
references: [AGENTS.md]

### How It Works

1. Resolves package → git repo URL via registry APIs (npm/PyPI/crates.io/GitHub API)
2. For npm: auto-detects installed version from lockfiles (package-lock, pnpm-lock, yarn.lock)
3. Shallow-clones at `v<version>` tag (falls back to `<version>`, then default branch)
4. Caches at `~/.opensrc/repos/<host>/<owner>/<repo>/<version>/`
5. Records in `~/.opensrc/sources.json` (atomic writes, corrupt-safe)

Env vars: `OPENSRC_HOME` (cache location), `GITHUB_TOKEN` / `GITLAB_TOKEN` / `BITBUCKET_TOKEN` (private repos).
