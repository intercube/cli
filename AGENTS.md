# Agent Guide

This repository contains the Intercube CLI, a Go command-line tool for
Intercube operational workflows such as onboarding, SSH, sync, organization
selection, site lookups, and Boundary-backed access.

## Project Snapshot

- **Language**: Go, module `github.com/intercube/cli`.
- **CLI framework**: Cobra.
- **Configuration**: Viper plus Intercube context configuration helpers.
- **Auth and integrations**: Clerk, Boundary API, inventory/API clients, rsync.
- **Entrypoint**: `cmd/intercube/main.go`.

## Repository Layout

- `cmd/`: Cobra commands and command-specific behavior.
- `cmd/intercube/`: executable entrypoint.
- `util/`: shared helpers for auth, app config, context detection, inventory, and configuration.
- `README.md`: user-facing behavior for onboarding, sync, and context-aware defaults.
- `CONTEXT-AWARE-CONFIG-PLAN.md`: design notes for context-aware configuration work.

## Working Rules

- Keep command behavior explicit and predictable in non-interactive contexts.
- Do not persist secrets unless the existing auth/config storage path already
  does so intentionally.
- Avoid logging passwords, Clerk tokens, Boundary tokens, API keys, private keys,
  or full config files.
- Preserve backward-compatible command flags and aliases unless the task
  explicitly changes the CLI contract.
- Match existing Cobra command patterns in `cmd/` before introducing new helpers.
- Keep context resolution precedence aligned with the README:
  flags, environment variables, active context config, then user defaults.
- In non-interactive mode, fail with a clear error instead of prompting.

## Common Commands

Format changed Go files:

```bash
gofmt -w <files>
```

Run tests:

```bash
go test ./...
```

Build the CLI:

```bash
go build ./cmd/intercube
```

Run a command locally:

```bash
go run ./cmd/intercube --help
```

## Testing Guidance

- Add tests for config resolution, target matching, and command behavior when the
  change affects decision logic.
- Prefer focused package tests over broad integration-style tests that need real
  Clerk, Boundary, SSH, or inventory services.
- For interactive flows, test the pure resolution/selection logic where
  possible and keep real prompt behavior thin.

## Cross-Repository Notes

- Backend API or inventory contract changes may require matching CLI updates.
- Ansible/AWX workflow changes may require CLI help text or command behavior
  updates if operators use the CLI to reach those systems.
- Dashboard changes usually should not affect this repo unless user-facing
  operational flows or shared API assumptions change.
