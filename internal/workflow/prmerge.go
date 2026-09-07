package workflow

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gitskill/gw/internal/decision"
	"github.com/gitskill/gw/internal/lock"
	"github.com/gitskill/gw/internal/redact"
	"github.com/gitskill/gw/internal/result"
)

// PRMerge merges the pull request with two-factor approval verification and lock guard.
func PRMerge(ctx context.Context, p Params) result.Envelope {
	branchOverride := ""
	if p.PRNumber > 0 {
		branchOverride = strconv.Itoa(p.PRNumber)
	}

	st, err := observeState(ctx, p, branchOverride)
	if err != nil {
		return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "PR_MERGE", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}

	d := decision.PRMerge(st, p.Policy)
	if d.Status != result.Success {
		var retryAfter *int
		if d.Status == result.Wait {
			retryAfter = intPtr(30)
		}
		return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
			d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, retryAfter, nil,
		), false)
	}

	// Preconditions and 2FA verified -> Acquire lock
	gitDir := filepath.Join(st.Repository.Root, ".git")
	l, reclaimed, err := lock.Acquire(gitDir, p.Policy.Lock.StaleAfter)
	if err != nil {
		if errors.Is(err, lock.ErrLocked) {
			return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
				result.Blocked, "REPO_LOCKED", "PR_MERGE", "Repository lock is held by another process.", nil, nil, nil, nil,
			), false)
		}
		return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
			result.CommandFailed, "LOCK_ACQUIRE_FAILED", "PR_MERGE", "Failed to acquire repository lock: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}
	defer l.Release()

	// Build gh pr merge command
	targetPR := strconv.Itoa(st.GitHub.PullRequest.Number)
	method := p.MergeMethod
	if method == "" {
		method = p.Policy.Merge.Method
	}
	if method == "" {
		method = "squash"
	}

	var mergeFlag string
	switch method {
	case "merge":
		mergeFlag = "--merge"
	case "rebase":
		mergeFlag = "--rebase"
	default:
		mergeFlag = "--squash"
	}

	mergeRes, err := p.Runner.Run(ctx, p.Dir, "gh", "pr", "merge", targetPR, mergeFlag, "--auto")
	if err != nil || mergeRes.ExitCode != 0 {
		// Try without --auto if auto-merge is not configured
		mergeRes, err = p.Runner.Run(ctx, p.Dir, "gh", "pr", "merge", targetPR, mergeFlag)
		if err != nil || mergeRes.ExitCode != 0 {
			return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
				result.CommandFailed, "GH_PR_MERGE_FAILED", "PR_MERGE", redact.Scrub(mergeRes.Stderr), nil, nil, nil, nil,
			), reclaimed)
		}
	}

	// Invalidate cache and verify
	invalidateCache(p)

	postSt, err := observeState(ctx, p, branchOverride)
	if err == nil && postSt.GitHub.PullRequest != nil && postSt.GitHub.PullRequest.State != "MERGED" {
		return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
			result.VerificationFailed, "POST_MERGE_UNMERGED", "PR_MERGE", "PR state is not MERGED following merge command execution.", nil, nil, nil, nil,
		), reclaimed)
	}

	return logAndBuildEnvelope(p, "pr-merge", result.NewEnvelope(
		result.Success, "MERGED", "PR_MERGE", fmt.Sprintf("Pull request #%s merged successfully.", targetPR), nil, postSt, nil, nil,
	), reclaimed)
}