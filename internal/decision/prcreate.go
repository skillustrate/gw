package decision

import (
	"fmt"

	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// PRCreateDecide determines whether PR creation may proceed.
func PRCreateDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.Repository.IsGitRepo {
		return Decision{
			Action: "PR_CREATE",
			Status: result.Blocked,
			Code:   "NOT_A_REPO",
			Reason: "Cannot create PR outside a Git repository.",
		}
	}

	if s.Branch.IsDefault {
		return Decision{
			Action: "PR_CREATE",
			Status: result.Blocked,
			Code:   "ON_DEFAULT_BRANCH",
			Reason: "Cannot create a pull request from the default base branch.",
		}
	}

	if s.GitHub.PullRequest != nil && s.GitHub.PullRequest.State == "OPEN" {
		return Decision{
			Action: "PR_CREATE",
			Status: result.Noop,
			Code:   "PR_ALREADY_EXISTS",
			Reason: fmt.Sprintf("Pull request #%d already exists for branch %s.", s.GitHub.PullRequest.Number, s.Repository.CurrentBranch),
		}
	}

	if !s.WorkingTree.Clean {
		return Decision{
			Action:     "PR_CREATE",
			Status:     result.Blocked,
			Code:       "WORKTREE_DIRTY",
			Reason:     "Working tree contains uncommitted changes.",
			NextAction: strPtr("gw prepare --json"),
		}
	}

	if !s.Branch.HasUpstream || s.Branch.Ahead > 0 {
		if !p.Push.Allow {
			return Decision{
				Action: "PR_CREATE",
				Status: result.PolicyDenied,
				Code:   "PUSH_DISALLOWED",
				Reason: "Branch has unpushed commits and push.allow is false.",
			}
		}
		return Decision{
			Action:     "PR_CREATE",
			Status:     result.ActionRequired,
			Code:       "PUSH_REQUIRED",
			Reason:     "Branch must be pushed before creating a pull request.",
			NextAction: strPtr("gw push --json"),
		}
	}

	if !p.PullRequest.Create {
		return Decision{
			Action: "PR_CREATE",
			Status: result.PolicyDenied,
			Code:   "PR_CREATE_DISALLOWED",
			Reason: "Pull request creation is disallowed by policy.",
		}
	}

	if !s.GitHub.Authenticated {
		return Decision{
			Action:     "PR_CREATE",
			Status:     result.AuthRequired,
			Code:       "GH_UNAUTHENTICATED",
			Reason:     "GitHub CLI is not authenticated.",
			NextAction: strPtr("gh auth login"),
		}
	}

	return Decision{
		Action: "PR_CREATE",
		Status: result.Success,
		Code:   "CREATE_PR",
		Reason: "Branch is clean, pushed, and ready for pull request creation.",
	}
}

var PRCreate Decider = PRCreateDecide