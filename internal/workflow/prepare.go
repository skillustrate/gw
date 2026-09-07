package workflow

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/gitskill/gw/internal/decision"
	"github.com/gitskill/gw/internal/lock"
	"github.com/gitskill/gw/internal/redact"
	"github.com/gitskill/gw/internal/result"
)

// Prepare executes the prepare workflow, making a commit only if commit.allow is true and message is provided.
func Prepare(ctx context.Context, p Params) result.Envelope {
	st, err := observeState(ctx, p, "")
	if err != nil {
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "PREPARE", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}

	d := decision.Prepare(st, p.Policy)

	// If decision is not COMMIT (e.g. CLEAN, PUSH_REQUIRED, COMMIT_OR_DISCARD_CHANGES)
	if d.Code != "COMMIT" {
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, nil, nil,
		), false)
	}

	// Decision is COMMIT. Check if caller supplied message.
	if p.CommitMessage == "" {
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			result.ActionRequired,
			"COMMIT_MESSAGE_REQUIRED",
			"PREPARE",
			"Working tree changes require a commit message (--message). The engine never invents commit messages.",
			strPtr("gw prepare --message \"<summary>\" --json"),
			st,
			nil,
			nil,
		), false)
	}

	// Acquire lock for mutation
	gitDir := filepath.Join(st.Repository.Root, ".git")
	l, reclaimed, err := lock.Acquire(gitDir, p.Policy.Lock.StaleAfter)
	if err != nil {
		if errors.Is(err, lock.ErrLocked) {
			return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
				result.Blocked, "REPO_LOCKED", "PREPARE", "Repository lock is held by another process.", nil, nil, nil, nil,
			), false)
		}
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			result.CommandFailed, "LOCK_ACQUIRE_FAILED", "PREPARE", "Failed to acquire repository lock: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}
	defer l.Release()

	// Execute: git add -A && git commit -m <msg>
	addRes, err := p.Runner.Run(ctx, p.Dir, "git", "add", "-A")
	if err != nil || addRes.ExitCode != 0 {
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			result.CommandFailed, "GIT_ADD_FAILED", "PREPARE", redact.Scrub(addRes.Stderr), nil, nil, nil, nil,
		), reclaimed)
	}

	commitRes, err := p.Runner.Run(ctx, p.Dir, "git", "commit", "-m", p.CommitMessage)
	if err != nil || commitRes.ExitCode != 0 {
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			result.CommandFailed, "GIT_COMMIT_FAILED", "PREPARE", redact.Scrub(commitRes.Stderr), nil, nil, nil, nil,
		), reclaimed)
	}

	// Invalidate cache and verify
	invalidateCache(p)

	postSt, err := observeState(ctx, p, "")
	if err != nil || !postSt.WorkingTree.Clean {
		return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
			result.VerificationFailed, "POST_COMMIT_DIRTY", "PREPARE", "Working tree is still dirty after commit.", nil, nil, nil, nil,
		), reclaimed)
	}

	return logAndBuildEnvelope(p, "prepare", result.NewEnvelope(
		result.Success, "COMMITTED", "PREPARE", "Changes staged and committed successfully.", nil, postSt, nil, nil,
	), reclaimed)
}