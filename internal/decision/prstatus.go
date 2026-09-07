package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// PRStatusDecide evaluates PR state reporting.
func PRStatusDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.GitHub.Available {
		return Decision{
			Action: "PR_STATUS",
			Status: result.ValidationFailed,
			Code:   "GH_UNAVAILABLE",
			Reason: "GitHub CLI is not installed or available on PATH.",
		}
	}

	if !s.GitHub.Authenticated {
		return Decision{
			Action:     "PR_STATUS",
			Status:     result.AuthRequired,
			Code:       "GH_UNAUTHENTICATED",
			Reason:     "GitHub CLI is not authenticated.",
			NextAction: strPtr("gh auth login"),
		}
	}

	if s.GitHub.PullRequest == nil {
		return Decision{
			Action: "PR_STATUS",
			Status: result.Noop,
			Code:   "NO_PR_FOUND",
			Reason: "No pull request found for current branch.",
		}
	}

	return Decision{
		Action: "PR_STATUS",
		Status: result.Success,
		Code:   "PR_STATUS_OBSERVED",
		Reason: "Pull request status retrieved successfully.",
	}
}

var PRStatus Decider = PRStatusDecide