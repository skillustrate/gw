package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// PRMergeDecide determines whether a pull request may be merged, enforcing two-factor approval.
func PRMergeDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.GitHub.Authenticated {
		return Decision{
			Action:     "PR_MERGE",
			Status:     result.AuthRequired,
			Code:       "GH_UNAUTHENTICATED",
			Reason:     "GitHub CLI is not authenticated.",
			NextAction: strPtr("gh auth login"),
		}
	}

	if s.GitHub.PullRequest == nil {
		return Decision{
			Action: "PR_MERGE",
			Status: result.Blocked,
			Code:   "NO_PR_FOUND",
			Reason: "No pull request found for current branch to merge.",
		}
	}

	pr := s.GitHub.PullRequest

	if pr.State == "MERGED" {
		return Decision{
			Action: "PR_MERGE",
			Status: result.Noop,
			Code:   "ALREADY_MERGED",
			Reason: "Pull request is already merged.",
		}
	}

	if pr.State == "CLOSED" {
		return Decision{
			Action: "PR_MERGE",
			Status: result.Blocked,
			Code:   "PR_CLOSED",
			Reason: "Pull request is closed and cannot be merged.",
		}
	}

	if pr.Mergeable != nil && !*pr.Mergeable {
		return Decision{
			Action:     "PR_MERGE",
			Status:     result.Conflict,
			Code:       "MERGE_CONFLICT",
			Reason:     "Pull request has merge conflicts that must be resolved.",
			NextAction: strPtr("Resolve merge conflicts on GitHub or rebase locally."),
		}
	}

	if p.Merge.RequireChecks {
		if pr.ChecksStatus == "pending" {
			return Decision{
				Action:     "PR_MERGE",
				Status:     result.Wait,
				Code:       "CHECKS_PENDING",
				Reason:     "Required CI checks are still in progress.",
				NextAction: strPtr("gw pr-status --json"),
			}
		}
		if pr.ChecksStatus == "failing" {
			return Decision{
				Action: "PR_MERGE",
				Status: result.Blocked,
				Code:   "CHECKS_FAILING",
				Reason: "Required CI checks are failing.",
			}
		}
	}

	if !p.Merge.Allow {
		return Decision{
			Action: "PR_MERGE",
			Status: result.PolicyDenied,
			Code:   "MERGE_DISALLOWED",
			Reason: "Merging is disallowed by policy.",
		}
	}

	// Two-factor approval rule (§14): Requires recorded GitHub review approval AND explicit --yes confirmation
	if p.Merge.RequireApproval && pr.ReviewDecision != "APPROVED" {
		return Decision{
			Action:     "PR_MERGE",
			Status:     result.HumanApprovalRequired,
			Code:       "REVIEW_APPROVAL_REQUIRED",
			Reason:     "Pull request requires approved code review on GitHub before merging.",
			NextAction: strPtr("Obtain reviewer approval on GitHub PR."),
		}
	}

	if !p.Approved {
		return Decision{
			Action:     "PR_MERGE",
			Status:     result.HumanApprovalRequired,
			Code:       "CLI_CONFIRMATION_REQUIRED",
			Reason:     "Merge requires explicit user confirmation (--yes flag).",
			NextAction: strPtr("gw pr-merge --json --yes"),
		}
	}

	return Decision{
		Action: "PR_MERGE",
		Status: result.Success,
		Code:   "MERGE",
		Reason: "Pull request is approved, checks passing, and authorized for merge.",
	}
}

var PRMerge Decider = PRMergeDecide