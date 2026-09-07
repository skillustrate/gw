package workflow

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gitskill/gw/internal/decision"
	"github.com/gitskill/gw/internal/lock"
	"github.com/gitskill/gw/internal/redact"
	"github.com/gitskill/gw/internal/result"
)

// PRCreate creates a pull request when observed preconditions and policy permit.
func PRCreate(ctx context.Context, p Params) result.Envelope {
	st, err := observeState(ctx, p, "")
	if err != nil {
		return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "PR_CREATE", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}

	d := decision.PRCreate(st, p.Policy)

	// If unpushed, invoke Push first if policy permits
	if d.Code == "PUSH_REQUIRED" && p.Policy.Push.Allow {
		pushEnv := Push(ctx, p)
		if pushEnv.Status != result.Success && pushEnv.Status != result.Noop {
			return pushEnv
		}

		// Re-observe and re-decide (§8 requirement: never assume success)
		st, err = observeState(ctx, p, "")
		if err != nil {
			return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
				result.CommandFailed, "OBSERVE_FAILED", "PR_CREATE", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
			), false)
		}
		d = decision.PRCreate(st, p.Policy)
	}

	if d.Status != result.Success {
		return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
			d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, nil, nil,
		), false)
	}

	// Acquire lock for mutation
	gitDir := filepath.Join(st.Repository.Root, ".git")
	l, reclaimed, err := lock.Acquire(gitDir, p.Policy.Lock.StaleAfter)
	if err != nil {
		if errors.Is(err, lock.ErrLocked) {
			return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
				result.Blocked, "REPO_LOCKED", "PR_CREATE", "Repository lock is held by another process.", nil, nil, nil, nil,
			), false)
		}
		return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
			result.CommandFailed, "LOCK_ACQUIRE_FAILED", "PR_CREATE", "Failed to acquire repository lock: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}
	defer l.Release()

	// Build gh pr create args
	args := []string{"pr", "create"}
	title := p.PRTitle
	if title == "" {
		title = st.Repository.CurrentBranch
	}
	args = append(args, "--title", title)

	body := p.PRBody
	if body == "" {
		body = "Automated pull request for branch " + st.Repository.CurrentBranch
	}
	args = append(args, "--body", body)

	base := p.BaseBranch
	if base == "" {
		base = p.Policy.BaseBranch
	}
	if base != "" {
		args = append(args, "--base", base)
	}

	if p.PRDraft || p.Policy.PullRequest.Draft {
		args = append(args, "--draft")
	}

	// Execute gh pr create
	createRes, err := p.Runner.Run(ctx, p.Dir, "gh", args...)
	if err != nil || createRes.ExitCode != 0 {
		return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
			result.CommandFailed, "GH_PR_CREATE_FAILED", "PR_CREATE", redact.Scrub(createRes.Stderr), nil, nil, nil, nil,
		), reclaimed)
	}

	// Invalidate cache and verify
	invalidateCache(p)

	postSt, err := observeState(ctx, p, "")
	if err != nil || postSt.GitHub.PullRequest == nil {
		prURL := strings.TrimSpace(createRes.Stdout)
		return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
			result.Success, "PR_CREATED", "PR_CREATE", "Pull request created successfully.", nil, map[string]any{
				"url": prURL,
			}, nil, nil,
		), reclaimed)
	}

	return logAndBuildEnvelope(p, "pr-create", result.NewEnvelope(
		result.Success, "PR_CREATED", "PR_CREATE", fmt.Sprintf("Pull request #%d created successfully.", postSt.GitHub.PullRequest.Number), nil, postSt, nil, nil,
	), reclaimed)
}