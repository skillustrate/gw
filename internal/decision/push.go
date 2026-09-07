package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// PushDecide determines whether the branch should be pushed and with what tracking.
func PushDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.Repository.IsGitRepo {
		return Decision{
			Action: "PUSH",
			Status: result.Blocked,
			Code:   "NOT_A_REPO",
			Reason: "Cannot push outside a Git repository.",
		}
	}

	if !p.Push.Allow {
		return Decision{
			Action: "PUSH",
			Status: result.PolicyDenied,
			Code:   "PUSH_DISALLOWED",
			Reason: "Pushing is disallowed by policy.",
		}
	}

	if s.Branch.HasUpstream {
		if s.Branch.Diverged {
			return Decision{
				Action:     "PUSH",
				Status:     result.Conflict,
				Code:       "BRANCH_DIVERGED",
				Reason:     "Local branch has diverged from upstream tracking branch.",
				NextAction: strPtr("gw sync --json"),
			}
		}
		if s.Branch.Ahead == 0 {
			return Decision{
				Action: "PUSH",
				Status: result.Noop,
				Code:   "ALREADY_PUSHED",
				Reason: "Branch is already synchronized with remote upstream.",
			}
		}
		if p.Push.RequireConfirmation && !p.Approved {
			return Decision{
				Action:     "PUSH",
				Status:     result.HumanApprovalRequired,
				Code:       "CLI_CONFIRMATION_REQUIRED",
				Reason:     "Pushing commits requires explicit user confirmation (--yes flag).",
				NextAction: strPtr("gw push --json --yes"),
			}
		}
		return Decision{
			Action: "PUSH",
			Status: result.Success,
			Code:   "PUSH",
			Reason: "Branch has local commits ready to push.",
		}
	}

	// No upstream branch exists
	if !p.Push.AllowSetUpstream {
		return Decision{
			Action: "PUSH",
			Status: result.PolicyDenied,
			Code:   "SET_UPSTREAM_DISALLOWED",
			Reason: "Establishing a new upstream tracking branch is disallowed by policy.",
		}
	}

	if p.Push.RequireConfirmation && !p.Approved {
		return Decision{
			Action:     "PUSH",
			Status:     result.HumanApprovalRequired,
			Code:       "CLI_CONFIRMATION_REQUIRED",
			Reason:     "Pushing commits with a new upstream tracking branch requires explicit user confirmation (--yes flag).",
			NextAction: strPtr("gw push --json --yes"),
		}
	}

	return Decision{
		Action: "PUSH",
		Status: result.Success,
		Code:   "PUSH_WITH_TRACKING",
		Reason: "Branch has unpublished local commits; ready to push with upstream tracking.",
	}
}

var Push Decider = PushDecide