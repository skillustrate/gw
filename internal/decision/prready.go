package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// PRReadyDecide answers whether the current branch is ready for a pull request in fixed precedence order.
func PRReadyDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.Repository.IsGitRepo {
		return Decision{
			Action: "PR_READY",
			Status: result.Blocked,
			Code:   "NOT_A_REPO",
			Reason: "Cannot create a PR outside a Git repository.",
		}
	}

	if s.Repository.Detached {
		return Decision{
			Action: "PR_READY",
			Status: result.Blocked,
			Code:   "DETACHED_HEAD",
			Reason: "Cannot create a PR from detached HEAD state.",
		}
	}

	if s.Branch.IsDefault {
		return Decision{
			Action: "PR_READY",
			Status: result.Blocked,
			Code:   "ON_DEFAULT_BRANCH",
			Reason: "Cannot create a PR from the default base branch.",
		}
	}

	if !s.WorkingTree.Clean {
		return Decision{
			Action:     "PR_READY",
			Status:     result.Blocked,
			Code:       "WORKTREE_DIRTY",
			Reason:     "Working tree contains uncommitted changes.",
			NextAction: strPtr("gw prepare --json"),
		}
	}

	if !s.Branch.HasUpstream || s.Branch.Ahead > 0 {
		return Decision{
			Action:     "PR_READY",
			Status:     result.ActionRequired,
			Code:       "NOT_PUSHED",
			Reason:     "Branch has unpushed commits.",
			NextAction: strPtr("gw push --json"),
		}
	}

	if s.GitHub.PullRequest != nil && s.GitHub.PullRequest.State == "OPEN" {
		return Decision{
			Action: "PR_READY",
			Status: result.Noop,
			Code:   "PR_ALREADY_EXISTS",
			Reason: "An open pull request already exists for this branch.",
		}
	}

	if !s.GitHub.Authenticated {
		return Decision{
			Action:     "PR_READY",
			Status:     result.AuthRequired,
			Code:       "GH_UNAUTHENTICATED",
			Reason:     "GitHub CLI is not authenticated.",
			NextAction: strPtr("gh auth login"),
		}
	}

	return Decision{
		Action: "PR_READY",
		Status: result.Success,
		Code:   "CREATE_PR",
		Reason: "Branch is clean, pushed, and ready for pull request creation.",
	}
}

var PRReady Decider = PRReadyDecide