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

// Push executes the branch push workflow guarded by repository lock.
func Push(ctx context.Context, p Params) result.Envelope {
	st, err := observeState(ctx, p, "")
	if err != nil {
		return logAndBuildEnvelope(p, "push", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "PUSH", err.Error(), nil, nil, nil, nil,
		), false)
	}

	d := decision.Push(st, p.Policy)
	if d.Status != result.Success {
		return logAndBuildEnvelope(p, "push", result.NewEnvelope(
			d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, nil, nil,
		), false)
	}

	// Acquire lock for mutation
	gitDir := filepath.Join(st.Repository.Root, ".git")
	l, reclaimed, err := lock.Acquire(gitDir, p.Policy.Lock.StaleAfter)
	if err != nil {
		if errors.Is(err, lock.ErrLocked) {
			return logAndBuildEnvelope(p, "push", result.NewEnvelope(
				result.Blocked, "REPO_LOCKED", "PUSH", "Repository lock is held by another process.", nil, nil, nil, nil,
			), false)
		}
		return logAndBuildEnvelope(p, "push", result.NewEnvelope(
			result.CommandFailed, "LOCK_ACQUIRE_FAILED", "PUSH", "Failed to acquire repository lock: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}
	defer l.Release()

	// Execute push command
	var pushRes struct {
		ExitCode int
		Stderr   string
	}

	if d.Code == "PUSH_WITH_TRACKING" {
		res, err := p.Runner.Run(ctx, p.Dir, "git", "push", "--set-upstream", "origin", st.Repository.CurrentBranch)
		if err != nil {
			return logAndBuildEnvelope(p, "push", result.NewEnvelope(
				result.CommandFailed, "PUSH_FAILED", "PUSH", redact.Scrub(err.Error()), nil, nil, nil, nil,
			), reclaimed)
		}
		pushRes.ExitCode = res.ExitCode
		pushRes.Stderr = res.Stderr
	} else {
		res, err := p.Runner.Run(ctx, p.Dir, "git", "push")
		if err != nil {
			return logAndBuildEnvelope(p, "push", result.NewEnvelope(
				result.CommandFailed, "PUSH_FAILED", "PUSH", redact.Scrub(err.Error()), nil, nil, nil, nil,
			), reclaimed)
		}
		pushRes.ExitCode = res.ExitCode
		pushRes.Stderr = res.Stderr
	}

	if pushRes.ExitCode != 0 {
		return logAndBuildEnvelope(p, "push", result.NewEnvelope(
			result.CommandFailed, "PUSH_FAILED", "PUSH", redact.Scrub(pushRes.Stderr), nil, nil, nil, nil,
		), reclaimed)
	}

	// Invalidate cache and verify
	invalidateCache(p)

	postSt, err := observeState(ctx, p, "")
	if err != nil || postSt.Branch.Ahead > 0 || !postSt.Branch.HasUpstream {
		return logAndBuildEnvelope(p, "push", result.NewEnvelope(
			result.VerificationFailed, "POST_PUSH_MISMATCH", "PUSH", "Verification failed: commits remain unpushed.", nil, nil, nil, nil,
		), reclaimed)
	}

	return logAndBuildEnvelope(p, "push", result.NewEnvelope(
		result.Success, "PUSHED", "PUSH", "Branch pushed to upstream successfully.", nil, postSt, nil, nil,
	), reclaimed)
}