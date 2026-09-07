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

// Sync synchronizes the local branch using fast-forward only strategy.
func Sync(ctx context.Context, p Params) result.Envelope {
	st, err := observeState(ctx, p, "")
	if err != nil {
		return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "SYNC", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}

	d := decision.Sync(st, p.Policy)
	if d.Status != result.Success {
		return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
			d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, nil, nil,
		), false)
	}

	// Acquire lock for mutation
	gitDir := filepath.Join(st.Repository.Root, ".git")
	l, reclaimed, err := lock.Acquire(gitDir, p.Policy.Lock.StaleAfter)
	if err != nil {
		if errors.Is(err, lock.ErrLocked) {
			return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
				result.Blocked, "REPO_LOCKED", "SYNC", "Repository lock is held by another process.", nil, nil, nil, nil,
			), false)
		}
		return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
			result.CommandFailed, "LOCK_ACQUIRE_FAILED", "SYNC", "Failed to acquire repository lock: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}
	defer l.Release()

	// Execute fetch & merge --ff-only
	fetchRes, err := p.Runner.Run(ctx, p.Dir, "git", "fetch", "origin")
	if err != nil || fetchRes.ExitCode != 0 {
		return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
			result.CommandFailed, "FETCH_FAILED", "SYNC", redact.Scrub(fetchRes.Stderr), nil, nil, nil, nil,
		), reclaimed)
	}

	mergeTarget := st.Branch.Upstream
	if p.BaseBranch != "" {
		mergeTarget = "origin/" + p.BaseBranch
	}

	mergeRes, err := p.Runner.Run(ctx, p.Dir, "git", "merge", "--ff-only", mergeTarget)
	if err != nil || mergeRes.ExitCode != 0 {
		return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
			result.Conflict, "FF_MERGE_FAILED", "SYNC", redact.Scrub(mergeRes.Stderr), strPtr("Rebase or resolve merge conflicts manually."), nil, nil, nil,
		), reclaimed)
	}

	// Invalidate cache and verify
	invalidateCache(p)

	postSt, err := observeState(ctx, p, "")
	if err != nil || postSt.Branch.Behind > 0 {
		return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
			result.VerificationFailed, "POST_SYNC_BEHIND", "SYNC", "Verification failed: branch is still behind upstream.", nil, nil, nil, nil,
		), reclaimed)
	}

	return logAndBuildEnvelope(p, "sync", result.NewEnvelope(
		result.Success, "SYNCHRONIZED", "SYNC", "Branch fast-forwarded successfully.", nil, postSt, nil, nil,
	), reclaimed)
}