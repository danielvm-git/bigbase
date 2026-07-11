# Command built from user-controlled sources (3 instances)

**Source:** GHS Code Scanning (CodeQL)
**Severity:** CRITICAL
**CWE:** CWE-78 (OS Command Injection)
**GitHub Alerts:** #5, #6, #7

## Description
CodeQL detected 3 instances where Go code builds OS commands using user-controlled input without proper sanitization. An attacker who controls input parameters could inject arbitrary commands.

## Recommendation
Replace raw command construction with `exec.Command` using a fixed command + safe arguments. Use allowlist-based validation for any user-controlled input that must influence the command.

## Status
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
