package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// PrepareDecide evaluates whether working tree changes require preparation before PR creation.
func PrepareDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.Repository.IsGitRepo {
		return Decision{
			Action: "PREPARE",
			Status: result.Blocked,
			Code:   "NOT_A_REPO",
			Reason: "Cannot prepare outside a Git repository.",
		}
	}

	if s.WorkingTree.Clean {
		if s.Branch.Ahead > 0 {
			return Decision{
				Action:     "PREPARE",
				Status:     result.ActionRequired,
				Code:       "PUSH_REQUIRED",
				Reason:     "Working tree is clean; unpushed commits exist.",
				NextAction: strPtr("gw push --json"),
			}
		}
		return Decision{
			Action: "PREPARE",
			Status: result.Noop,
			Code:   "CLEAN",
			Reason: "Working tree is clean and synchronized.",
		}
	}

	// Working tree is dirty
	if !p.Commit.Allow {
		return Decision{
			Action:     "PREPARE",
			Status:     result.ActionRequired,
			Code:       "COMMIT_OR_DISCARD_CHANGES",
			Reason:     "Working tree contains uncommitted changes and commit.allow is false.",
			NextAction: strPtr("Commit or discard working tree changes."),
		}
	}

	if p.Commit.RequireConfirmation && !p.Approved {
		return Decision{
			Action:     "PREPARE",
			Status:     result.HumanApprovalRequired,
			Code:       "CLI_CONFIRMATION_REQUIRED",
			Reason:     "Committing changes requires explicit user confirmation (--yes flag).",
			NextAction: strPtr("gw prepare --json --yes --message \"<summary>\""),
		}
	}

	return Decision{
		Action: "PREPARE",
		Status: result.Success,
		Code:   "COMMIT",
		Reason: "Working tree contains changes ready to commit.",
	}
}

var Prepare Decider = PrepareDecide