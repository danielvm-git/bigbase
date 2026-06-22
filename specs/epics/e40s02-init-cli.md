# Story e40s02: bigbase init CLI and --manifest deploy flag

**type:** feat
**context:** infra
**BCPS:** 1

## Context

Add a new `bigbase init` command to the CLI that detects the framework in a repository and writes a sensible, default `bigbase.yaml` manifest. Also add a `--manifest` flag to the `bigbase deploy` command allowing deployments to use a specific manifest file name/path other than the default `bigbase.yaml`.

## Module Zoom-Out

### main.go (CLI Command Parser)
- **Purpose:** Parse and execute CLI commands like `serve`, `deploy`, `status`, `backup`, `restore`, `migrate`.
- **Callers:** User CLI execution.
- **Contracts:** CLI commands and their corresponding flag structures.

### components/deploy/deploy.go (Deploy Component)
- **Purpose:** Run, build, and manage deployments.
- **Callers:** REST API (`HandleCreate`/`HandleList`), CLI commands (`deploy`), sites controller.
- **Contracts:** Database representation of deployments, including triggering build and run execution.

### components/deploy/manifest.go (Manifest Parser)
- **Purpose:** Load, validate, and convert the YAML manifest into configuration settings.
- **Callers:** `Deploy.runDeployment`.
- **Contracts:** `LoadManifest(dir string) (*Manifest, error)` -> we will extend it to `LoadManifestPath(dir, manifestPath string) (*Manifest, error)`.

## Steps

### 1. Implement framework detection and YAML generation in `components/deploy/manifest.go`
**→ verify:** `go test ./components/deploy/ -run TestInitManifest -v`

- Implement `InitManifest(dir string) error` (or similar helper) that:
  - Detects the framework using package files:
    - If `package.json` exists:
      - Read and parse it. If `dependencies` or `devDependencies` contains `@sveltejs/kit` -> framework: `sveltekit`, build command: `npm run build`, output: `build/`, start command: `node build/index.js`, port: `3000`.
      - Otherwise -> framework: `node`, build command: `npm run build` (if script exists, else `echo build`), start command: start script (if exists) or `node index.js`, port: `3000`.
    - If `go.mod` exists -> framework: `go`, build command: `go build -o app .`, start command: `./app`, port: `8080`.
    - If `main.py` or `app.py` exists -> framework: `python`, build command: `pip install -r requirements.txt` (if exists), start command: `python main.py` or `python app.py`, port: `8000`.
    - Otherwise -> framework: `static`, build command: `echo static`, start command: `echo static`, port: `3000`.
  - Writes the generated default `bigbase.yaml` file to the specified directory.

### 2. Implement the `bigbase init` command in `main.go`
**→ verify:** `go run . init --repo /tmp/test-repo && test -f /tmp/test-repo/bigbase.yaml`

- Add `init` subcommand parsing to `main.go`.
- Accepts `--repo` flag, defaulting to current working directory.
- Runs framework detection and writes `bigbase.yaml`.
- Returns an error if the directory does not exist or if `bigbase.yaml` already exists (to prevent accidental overwrites).

### 3. Add `manifest_path` support to Database, Trigger, and deploy API
**→ verify:** `go test ./components/deploy/ -run TestManifestPathPersistence -v`

- Add `manifest_path TEXT NOT NULL DEFAULT ''` to the `deployments` database schema.
- Add database migration `ensureManifestPathColumn()` to `components/deploy/deploy.go`.
- Update `Deployment` struct to include `ManifestPath string `json:"manifest_path"``.
- Update `Trigger` signature to accept `manifestPath string`.
- Update `LoadManifest` usage to `LoadManifestPath(buildDir, deploy.ManifestPath)`.
- Update all internal `Trigger` callers (in `main.go`, `components/mcp/mcp.go`, `components/deploy/deploy_test.go`, and `components/deploy/samples.go`).

### 4. Implement `--manifest` flag in deploy CLI and verify merge order
**→ verify:** `go test ./components/deploy/ -run TestManifestFlags -v`

- Update `runDeployCmd()` in `main.go` to accept `--manifest` flag.
- Pass `manifest_path` inside the request body of the POST `/api/deploy` payload.
- In `deploy.go` build flow, ensure the merge order: CLI flags > manifest > auto-detection.

## Verification Script (Manual)

1. Create a dummy SvelteKit repository:
   ```bash
   mkdir -p /tmp/svelte-repo
   echo '{"dependencies": {"@sveltejs/kit": "^2.0.0"}}' > /tmp/svelte-repo/package.json
   ```
2. Initialize manifest:
   ```bash
   go run . init --repo /tmp/svelte-repo
   ```
3. Verify `bigbase.yaml` was created at `/tmp/svelte-repo/bigbase.yaml` with framework set to `sveltekit` and build/start commands properly configured.
4. Verify server builds and runs successfully:
   ```bash
   go test ./...
   ```

## Out of scope

- Initializing complex env templates beyond the default structure.
- Interactive prompts asking the user to override detected framework (non-interactive behavior is required by automated suites).

## Risks

- **Overwriting existing manifests**: Mitigation: Check if `bigbase.yaml` exists and fail/return early unless forced.
