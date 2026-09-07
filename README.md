# `gw` — GitHub-Aware Deterministic Git Workflow Engine

[![CI](https://github.com/gitskill/gw/actions/workflows/ci.yml/badge.svg)](https://github.com/gitskill/gw/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](go.mod)

**`gw`** is a deterministic, token-efficient Git and GitHub workflow engine designed for multi-model AI coding agents (Antigravity, Gemini CLI, Claude Code, Cursor, Codex, etc.) and developer tooling.

Instead of issuing sequences of raw, verbose, and unpredictable shell commands (`git status`, `git add`, `git commit`, `git push`, `gh pr create`, `gh pr checks`), AI agents invoke `gw` as a unified skill. `gw` observes repository state, computes deterministic decisions in pure code, executes guarded mutations, and returns compact JSON envelopes.

---

## 🚀 Key Highlights & Architectural Advantages

- **⚡ Token-Efficient (~80% Reduction):** Replaces verbose terminal dumps with single-call, schema-validated JSON envelopes.
- **🛡️ Safe-by-Default Policy Engine:** Disallows accidental commits, unauthorized pushes to base branches, and unapproved merges out-of-the-box.
- **🔒 Two-Factor Merge Safety:** Merging a PR requires both recorded GitHub code review approval (`APPROVED`) and explicit confirmation (`--yes`).
- **✋ Confirmation Gate on Mutations:** `prepare`, `push`, and `sync` each require explicit confirmation (`--yes`) by default (`*.require_confirmation`), enforced in the decision layer — not just prompted for by the calling agent.
- **🛑 Mutex Concurrency Guard:** Mutating operations acquire `.git/gw.lock` with automatic PID-liveness and age-based stale lock reclamation.
- **🛡️ Untrusted Repo Boundary:** Repo-local `.gw.yml` files are ignored by default unless explicitly enabled via `--allow-repo-config`.
- **🔑 Credential Redaction:** Automatic defense-in-depth sanitization of GitHub PATs, OAuth tokens, and Authorization headers in all logs and error streams.
- **🤖 Universal AI Skill Integration:** Out-of-the-box skills for Antigravity, Gemini, Claude, and OpenAI harnesses.

---

## 🔄 The `gw` Workflow Lifecycle

Every mutating workflow executes a strict, 4-phase lifecycle:

```mermaid
flowchart LR
    A["1. OBSERVE\n(Git + GitHub CLI)"] --> B["2. DECIDE\n(Pure Logic + Policy)"]
    B --> C["3. EXECUTE\n(Lock Guarded Mutation)"]
    C --> D["4. VERIFY\n(Post-State Validation)"]
    D --> E["5. ENVELOPE\n(JSON Response)"]
```

1. **OBSERVE:** Gathers unified Git status and GitHub PR/checks metadata via single-call GraphQL batching with in-process read caching.
2. **DECIDE:** Evaluates preconditions against user policy via pure, side-effect-free decision functions.
3. **EXECUTE:** Executes minimal necessary mutation while holding `.git/gw.lock`.
4. **VERIFY:** Re-observes the repository to prove the mutation succeeded before reporting `SUCCESS`.
5. **ENVELOPE:** Emits a structured JSON envelope with actionable `status`, `code`, `reason`, and `next_action`.

---

## 📋 Key Workflow Protocol for AI Agents

When integrated into coding harnesses, agents follow this strict protocol:

1. **Auto-Invocation:** The agent invokes `gw` whenever it needs to inspect, stage, commit, push, sync, create PRs, check CI, or merge.
2. **User Transparency Gate:**
   - **Read-Only Commands** (`inspect`, `pr-ready`, `pr-status`, `doctor`): Executed immediately.
   - **Mutating Commands** (`prepare`, `push`, `sync`, `pr-create`, `pr-merge`): **The agent must list planned operations (branch, files, PR title, merge method) and obtain user confirmation before executing.**
3. **Deterministic Envelope Handling:**
   - On `SUCCESS` / `NOOP`, the agent continues work using `data`.
   - On non-success (`ACTION_REQUIRED`, `BLOCKED`, `CONFLICT`, `WAIT`, `POLICY_DENIED`, `HUMAN_APPROVAL_REQUIRED`), the agent reports `reason` and follows `next_action` rather than improvising raw commands.

---

## 🛠️ Supported Workflows

| Command | Type | Description |
|---|---|---|
| `gw inspect` | Read-only | Observes and normalizes current Git repository and GitHub PR state. |
| `gw prepare` | Mutating (Gated) | Stages and commits changes if `commit.allow: true` and `--message` is provided. |
| `gw push` | Mutating | Pushes branch to remote with `--set-upstream` tracking if needed. |
| `gw sync` | Mutating | Fast-forward only synchronization (`git merge --ff-only`) with upstream. |
| `gw pr-ready` | Read-only | Evaluates branch readiness for pull request creation in fixed precedence. |
| `gw pr-create` | Mutating | Creates a pull request after verifying pushed state and clean working tree. |
| `gw pr-status` | Read-only | Retrieves normalized PR mergeability, CI status rollup, and review state. |
| `gw pr-merge` | Mutating (Guarded) | Merges PR after verifying 2FA (GitHub approval + `--yes`) and passing CI checks. |
| `gw doctor` | Diagnostic | Validates system environment (`git`, `gh`, authentication status). |

---

## 📦 JSON Envelope Contract

All commands support `--json` output formatted according to schema version `1`:

```json
{
  "schema_version": 1,
  "status": "SUCCESS",
  "code": "PR_CREATED",
  "action": "CREATE_PR",
  "reason": "Working tree clean, branch pushed, no existing PR.",
  "next_action": null,
  "data": {
    "pr": {
      "number": 142,
      "url": "https://github.com/owner/repo/pull/142"
    }
  },
  "retry_after_hint": null,
  "debug": null
}
```

### Status Taxonomy & Exit Codes

| Status | Process Exit Code | Meaning |
|---|---|---|
| `SUCCESS` | `0` | Action completed and post-mutation state verified. |
| `NOOP` | `0` | Repository is already in the desired state (e.g. already pushed, already merged). |
| `WAIT` | `0` | Asynchronous operation in progress (CI checks running); check `retry_after_hint`. |
| `ACTION_REQUIRED` | `1` | Caller intervention required (e.g. unpushed commits, uncommitted changes). |
| `BLOCKED` | `1` | Precondition failed (e.g. dirty working tree, detached HEAD, repo locked). |
| `AUTH_REQUIRED` | `1` | GitHub CLI unauthenticated or insufficient token scopes. |
| `CONFLICT` | `1` | Git divergence or PR merge conflicts detected; manual resolution needed. |
| `POLICY_DENIED` | `1` | Operation forbidden by configured policy. |
| `HUMAN_APPROVAL_REQUIRED` | `1` | High-risk mutation requires reviewer approval or `--yes` confirmation. |
| `VALIDATION_FAILED` | `1` or `2` | Flag usage error (`2`) or environment unsupported (`1`). |
| `COMMAND_FAILED` | `1` | Internal binary execution failed. |
| `VERIFICATION_FAILED` | `1` | Post-mutation verification check did not match expected outcome. |

---

## ⚙️ Configuration

Policy is configured in `~/.config/gw/config.yaml` or via CLI flags:

```yaml
version: 1
base_branch: main

commit:
  allow: false                 # Disallows automated commits by default
  require_confirmation: true   # Requires explicit --yes before committing

push:
  allow: true                  # Allows push to feature branches
  allow_set_upstream: true     # Allows setting remote tracking branch
  require_confirmation: true   # Requires explicit --yes before pushing

pull_request:
  create: true
  draft: false

merge:
  allow: false                 # Merges disabled by default
  require_checks: true         # Blocks merge if CI checks fail/pending
  require_approval: true       # Requires GitHub review approval before merge

sync:
  strategy: ff-only            # Fast-forward only (no accidental merge commits)
  require_confirmation: true   # Requires explicit --yes before syncing

lock:
  stale_after: 10m             # Automatically reclaims locks older than 10 minutes
```

## 📦 Installation & Setup

### 1. Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Git (2.30+)](https://git-scm.com/)
- [GitHub CLI (`gh`)](https://cli.github.com/) authenticated via `gh auth login`

```bash
# Verify prerequisites
git --version
gh auth status
go version
```

### 2. Install the `gw` Binary

#### Option A: Quick Install via `go install` (Recommended)
```bash
go install github.com/gitskill/gw/cmd/gw@latest
```
> [!NOTE]
> Ensure your Go binary directory (`$GOPATH/bin` or `~/go/bin`, or `%USERPROFILE%\go\bin` on Windows) is in your system `PATH`.

#### Option B: Build from Source
```bash
# Clone repository
git clone https://github.com/gitskill/gw.git
cd gw

# Build binary
go build -trimpath -ldflags "-s -w" -o bin/gw ./cmd/gw

# (Optional) Move to global PATH on Linux / macOS:
sudo cp bin/gw /usr/local/bin/

# On Windows: Add the bin/ directory to your PATH environment variable
```

#### Option C: Using Makefile
```bash
make build
# Binary created at bin/gw
```

### 3. Verify Installation
Run `gw doctor` to validate that `git`, `gh`, and authentication status are ready:
```bash
gw doctor

# Or format as a JSON envelope:
gw doctor --json
```

---

## 🤖 Installing the AI Skill

`gw` includes standard skill definitions ready to be installed in your AI coding agent configuration directories:

### 1. Google Antigravity / Gemini CLI

- **Workspace-Level (Current Project):**
  ```bash
  mkdir -p .gemini/skills/git-workflow
  cp skill/git-workflow-engine/SKILL.md .gemini/skills/git-workflow/SKILL.md
  ```
- **Global-Level (All Projects):**
  - **Linux / macOS:**
    ```bash
    mkdir -p ~/.gemini/skills/git-workflow
    cp skill/git-workflow-engine/SKILL.md ~/.gemini/skills/git-workflow/SKILL.md
    ```
  - **Windows (PowerShell):**
    ```powershell
    New-Item -ItemType Directory -Force "$HOME\.gemini\skills\git-workflow"
    Copy-Item "skill\git-workflow-engine\SKILL.md" "$HOME\.gemini\skills\git-workflow\SKILL.md"
    ```

### 2. Claude Code

- **Workspace-Level:**
  ```bash
  mkdir -p .claude/skills/git-workflow
  cp skill/git-workflow-engine/SKILL.md .claude/skills/git-workflow/SKILL.md
  ```
- **Global-Level:**
  ```bash
  mkdir -p ~/.claude/skills/git-workflow
  cp skill/git-workflow-engine/SKILL.md ~/.claude/skills/git-workflow/SKILL.md
  ```

### 3. Agent Frameworks & Subagent Squads

- Copy to the standard `.agents/skills` directory:
  ```bash
  mkdir -p .agents/skills/git-workflow
  cp skill/git-workflow-engine/SKILL.md .agents/skills/git-workflow/SKILL.md
  ```

### 4. Cursor / Codex / Windsurf / Custom Agents

- Reference `skill/git-workflow-engine/SKILL.md` directly in your system prompt, `.cursorrules`, or `.cursor/rules`:
  ```markdown
  When performing Git operations, branch synchronization, or Pull Request management, invoke the `gw` CLI tool and follow the protocol specified in skill/git-workflow-engine/SKILL.md.
  ```

---

## 💡 How to Use

### A. Using with AI Coding Agents

When the skill is loaded, your AI agent automatically intercepts Git/GitHub tasks and executes deterministic `gw` workflows.

#### Conversational Prompt Examples

| User Prompt / Goal | Agent Action | What `gw` Does |
|---|---|---|
| *"What is our git status and is PR open?"* | `gw inspect --json` | Observes unified status, active branch, ahead/behind commits, and PR details. |
| *"Is this branch ready for a pull request?"* | `gw pr-ready --json` | Validates clean tree, pushed commits, base divergence, and existing PR state. |
| *"Stage and commit these changes with message 'fix: token refresh'"* | `gw prepare --message "..." --yes --json` | Previews planned commit, obtains confirmation, stages and commits. |
| *"Push my branch to origin"* | `gw push --push-set-upstream --yes --json` | Pushes branch and configures upstream tracking if needed. |
| *"Sync current branch with main"* | `gw sync --yes --json` | Fast-forward syncs (`--ff-only`) with upstream to prevent accidental merge bubbles. |
| *"Create a pull request for this feature"* | `gw pr-create --title "..." --yes --json` | Verifies clean working tree & pushed state, then creates GitHub PR. |
| *"Check PR status and CI checks"* | `gw pr-status --json` | Retrieves rollup of CI check runs, review status, and mergeability. |
| *"Merge this pull request"* | `gw pr-merge --merge-method squash --yes --json` | Enforces 2FA (approved review + explicit confirmation) and green CI checks before merging. |

#### AI Agent Safety Protocol
1. **Zero Prompt for Read-Only:** Read-only operations (`inspect`, `pr-ready`, `pr-status`, `doctor`) execute immediately via `--json`.
2. **Mandatory Confirmation for Mutations:** Before running mutating commands (`prepare`, `push`, `sync`, `pr-create`, `pr-merge`), the agent presents planned actions (action type, branches, affected files/PR title) and requests confirmation before running with `--yes`.
3. **Structured Response Handling:** The agent inspects `status`, `code`, and follows `next_action` on non-success rather than guessing raw shell commands.

---

### B. Using Directly from Terminal (CLI)

`gw` can also be run manually as a developer CLI:

```bash
# 1. Run diagnostic environment check
gw doctor

# 2. Inspect repository and PR status
gw inspect

# 3. Stage & commit modified files (overriding default commit policy with explicit confirmation)
gw prepare --message "feat: implement caching layer" --commit-allow --yes

# 4. Push branch to remote with upstream tracking
gw push --push-set-upstream --yes

# 5. Check if branch is ready for a PR
gw pr-ready

# 6. Create a Pull Request
gw pr-create --title "feat: implement caching layer" --body "Adds in-memory caching"

# 7. Check PR mergeability & CI status
gw pr-status

# 8. Fast-forward sync branch with upstream
gw sync --yes

# 9. Guarded PR Merge (verifies GitHub code review approval + passing CI + --yes)
gw pr-merge --merge-method squash --merge-allow --yes
```

---

## 🧪 Testing & Development

```bash
# Run unit and core tests
go test ./... -v

# Run vendor neutrality lint
go test ./internal/vendorcheck/... -v

# Run integration & E2E tests (requires git)
go test ./test/... -v
```

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).