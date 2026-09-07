# End-to-End Tests (Opt-In)

These tests run real GitHub mutations against a dedicated disposable test repository.
They are never executed during standard `go test ./...` runs.

## Prerequisites

1. `gh` CLI installed and authenticated (`gh auth login`).
2. A disposable GitHub test repository configured with permissions.
3. Environment variable `GW_E2E_REPO` set to the absolute path of the local clone.

## Running

```bash
GW_E2E_REPO=/path/to/disposable-repo go test -tags e2e ./test/e2e/... -v
```