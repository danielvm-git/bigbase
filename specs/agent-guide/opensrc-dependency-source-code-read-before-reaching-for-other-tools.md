# opensrc — Dependency Source Code (READ BEFORE reaching for other tools)

source: AGENTS.md
references: [AGENTS.md]

## opensrc — Dependency Source Code (READ BEFORE reaching for other tools)

`opensrc` fetches and caches the actual source code of any dependency from npm, PyPI, crates.io, or GitHub/GitLab/Bitbucket at `~/.opensrc/`. It resolves registry metadata → shallow-clones at the correct version tag → gives you a local filesystem path. **Whenever your question is about what a library DOES internally, opensrc is the right tool.**
