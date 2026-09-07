---
name: git-workflow-engine
description: Deterministic Git/GitHub workflow engine (gw). Use for committing or staging local changes, pushing or syncing a branch, creating a pull request, checking PR or CI status, or merging a pull request — instead of running raw git/gh commands.
---

# git-workflow-engine

Deterministic, token-efficient Git and GitHub workflow automation.

## Key Workflow Protocol

### 1. Auto-Invocation Boundary
Whenever you need to inspect repository state, stage/commit changes, push a branch, sync upstream, create a PR, check CI status, or merge a PR — invoke `gw` instead of executing raw shell `git` or `gh` commands.

### 2. User Permission & Transparency Gate
- **Read-Only Operations** (`inspect`, `pr-ready`, `pr-status`, `doctor`):
  Execute immediately via `gw <command> --json` to observe state without mutation.
- **Mutating Operations** (`prepare`, `push`, `sync`, `pr-create`, `pr-merge`):
  **Always list the planned operations to the user and obtain confirmation before executing**:
  ```markdown
  I will perform the following Git/GitHub operations:
  - **Action**: <prepare / push / sync / pr-create / pr-merge>
  - **Branch**: <current-branch> -> <target-branch>
  - **Details**: <files to commit / PR title / merge strategy>

  May I proceed?
  ```

### 3. Execution & Envelope Contract
1. Match the requested outcome to a command: `inspect`, `prepare`, `push`, `sync`, `pr-ready`, `pr-create`, `pr-status`, `pr-merge`.
2. Execute `gw <command> --json` (with necessary flags like `--yes`, `--message`, `--title`).
3. Read the returned JSON envelope's `status`, `code`, `reason`, and `next_action`.
4. On `SUCCESS` or `NOOP`, proceed using the payload in `data`.
5. On any other status (`ACTION_REQUIRED`, `BLOCKED`, `CONFLICT`, `WAIT`, `POLICY_DENIED`, `HUMAN_APPROVAL_REQUIRED`), report `reason` to the user and follow `next_action` as the complete plan — not a starting point for improvised raw git commands.
6. If `gw` is not on PATH or `gw doctor --json` reports `VALIDATION_FAILED`, inform the user and fall back to manual git/gh commands for this session.

### 4. Two-Factor Safety & Guardrails
- **Merge Safety**: Merging a PR requires both recorded GitHub code review approval (`APPROVED`) and explicit confirmation (`--yes`).
- **Confirmation Gate**: `prepare`, `push`, and `sync` return `HUMAN_APPROVAL_REQUIRED` (`CLI_CONFIRMATION_REQUIRED`) unless invoked with `--yes` — this is enforced by `gw` itself, not only by this skill's own confirmation step, so retry the exact `next_action` command with `--yes` after the user confirms.
- **Concurrency Guard**: All mutations are protected by `.git/gw.lock`.
- **Fast-Forward Only**: Synchronization strictly enforces `ff-only` by default and never forces merges or auto-resolves conflicts.