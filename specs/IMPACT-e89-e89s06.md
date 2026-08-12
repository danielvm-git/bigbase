type: impact-assessment
context: infra
epic: e89
story: e89s06
mode: lightweight

# Impact Assessment: e89s06

## Target

Extend the single EnvResolver seam to Project and Site compatibility secrets, replace
runtime legacy fetching, enforce scope precedence, redact logs, and provide explicit
resumable legacy migration.

## Module purpose, callers, contracts

- Purpose: `components/deploy` owns build/runtime process environment resolution,
  child-process injection, restart/resume, and deployment diagnostics; `backup` owns
  migration/restore tooling.
- Callers: buildApp, startApp, orchestrator resume, rollback, health/diagnostic paths,
  supported runtime child processes, and the composition root.
- Contracts: platform → manifest → Project → Site compatibility → reserved values;
  Site wins on collision; build-only values never reach runtime; static serving receives
  no secrets; protected decryption errors fail closed; migration format is explicit,
  resumable, and value-free; s01 redaction seam remains authoritative.

## Impact and risks

Critical/high. This changes shared deployment behavior and touches runtime processes,
logs, backup migration, and restart recovery. It runs parallel with s04 only because
its branch is restricted to Deploy/backup and consumes the frozen SecretManager seam.

## Coverage

Scenarios: SC-e89s06-P0-01 through SC-e89s06-P0-05 and SC-e89s06-P1-06.

## Recommended action

Start after s03 and s01 acceptance. Do not reimplement s01 runtime failure/redaction
logic or edit `main.go` in the story branch; coordinator wires the composition root.
