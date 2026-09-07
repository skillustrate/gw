package decision

import (
	"testing"

	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
)

func TestDecision_SyncDecide(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		s := baseRepoState()
		s.Repository.IsGitRepo = false
		d := SyncDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "NOT_A_REPO" {
			t.Errorf("got %+v, want BLOCKED(NOT_A_REPO)", d)
		}
	})

	t.Run("operation in progress", func(t *testing.T) {
		s := baseRepoState()
		op := "merge"
		s.WorkingTree.InProgressOperation = &op
		d := SyncDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "OPERATION_IN_PROGRESS" {
			t.Errorf("got %+v, want BLOCKED(OPERATION_IN_PROGRESS)", d)
		}
	})

	t.Run("branch diverged", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Diverged = true
		d := SyncDecide(s, policy.Default())
		if d.Status != result.Conflict || d.Code != "BRANCH_DIVERGED" {
			t.Errorf("got %+v, want CONFLICT(BRANCH_DIVERGED)", d)
		}
	})

	t.Run("already synchronized (behind == 0)", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Behind = 0
		d := SyncDecide(s, policy.Default())
		if d.Status != result.Noop || d.Code != "ALREADY_SYNCHRONIZED" {
			t.Errorf("got %+v, want NOOP(ALREADY_SYNCHRONIZED)", d)
		}
	})

	t.Run("behind with dirty working tree", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Behind = 3
		s.WorkingTree.Clean = false
		d := SyncDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "WORKTREE_DIRTY" {
			t.Errorf("got %+v, want BLOCKED(WORKTREE_DIRTY)", d)
		}
	})

	t.Run("behind with clean working tree requires confirmation", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Behind = 3
		s.WorkingTree.Clean = true
		d := SyncDecide(s, policy.Default())
		if d.Status != result.HumanApprovalRequired || d.Code != "CLI_CONFIRMATION_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(CLI_CONFIRMATION_REQUIRED)", d)
		}
	})

	t.Run("behind with clean working tree and confirmed", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Behind = 3
		s.WorkingTree.Clean = true
		p := policy.Default()
		p.Approved = true
		d := SyncDecide(s, p)
		if d.Status != result.Success || d.Code != "SYNC_FF_ONLY" {
			t.Errorf("got %+v, want SUCCESS(SYNC_FF_ONLY)", d)
		}
	})

	t.Run("behind with clean working tree and confirmation not required", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Behind = 3
		s.WorkingTree.Clean = true
		p := policy.Default()
		p.Sync.RequireConfirmation = false
		d := SyncDecide(s, p)
		if d.Status != result.Success || d.Code != "SYNC_FF_ONLY" {
			t.Errorf("got %+v, want SUCCESS(SYNC_FF_ONLY)", d)
		}
	})
}
