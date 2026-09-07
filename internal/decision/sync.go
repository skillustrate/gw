package decision

import (
	"fmt"

	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// SyncDecide determines whether branch synchronization is possible and safe.
func SyncDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.Repository.IsGitRepo {
		return Decision{
			Action: "SYNC",
			Status: result.Blocked,
			Code:   "NOT_A_REPO",
			Reason: "Cannot sync outside a Git repository.",
		}
	}

	if s.WorkingTree.InProgressOperation != nil {
		return Decision{
			Action: "SYNC",
			Status: result.Blocked,
			Code:   "OPERATION_IN_PROGRESS",
			Reason: fmt.Sprintf("A %s operation is currently in progress.", *s.WorkingTree.InProgressOperation),
		}
	}

	if s.Branch.Diverged {
		return Decision{
			Action:     "SYNC",
			Status:     result.Conflict,
			Code:       "BRANCH_DIVERGED",
			Reason:     "Branch has diverged from upstream; automatic fast-forward sync is not possible.",
			NextAction: strPtr("Resolve divergence manually or rebase before syncing."),
		}
	}

	if s.Branch.Behind == 0 {
		return Decision{
			Action: "SYNC",
			Status: result.Noop,
			Code:   "ALREADY_SYNCHRONIZED",
			Reason: "Branch is already up-to-date with upstream/base.",
		}
	}

	// Behind > 0
	if !s.WorkingTree.Clean {
		return Decision{
			Action:     "SYNC",
			Status:     result.Blocked,
			Code:       "WORKTREE_DIRTY",
			Reason:     "Cannot sync with uncommitted working tree changes.",
			NextAction: strPtr("gw prepare --json"),
		}
	}

	if p.Sync.RequireConfirmation && !p.Approved {
		return Decision{
			Action:     "SYNC",
			Status:     result.HumanApprovalRequired,
			Code:       "CLI_CONFIRMATION_REQUIRED",
			Reason:     "Synchronizing the branch requires explicit user confirmation (--yes flag).",
			NextAction: strPtr("gw sync --json --yes"),
		}
	}

	return Decision{
		Action: "SYNC",
		Status: result.Success,
		Code:   "SYNC_FF_ONLY",
		Reason: "Branch is behind upstream; fast-forward sync possible.",
	}
}

var Sync Decider = SyncDecide