package decision

import (
	"testing"

	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
)

func TestDecision_PrepareDecide(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		s := baseRepoState()
		s.Repository.IsGitRepo = false
		d := PrepareDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "NOT_A_REPO" {
			t.Errorf("got %+v, want BLOCKED(NOT_A_REPO)", d)
		}
	})

	t.Run("clean with unpushed commits", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = true
		s.Branch.Ahead = 1
		d := PrepareDecide(s, policy.Default())
		if d.Status != result.ActionRequired || d.Code != "PUSH_REQUIRED" {
			t.Errorf("got %+v, want ACTION_REQUIRED(PUSH_REQUIRED)", d)
		}
	})

	t.Run("clean and synchronized", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = true
		s.Branch.Ahead = 0
		d := PrepareDecide(s, policy.Default())
		if d.Status != result.Noop || d.Code != "CLEAN" {
			t.Errorf("got %+v, want NOOP(CLEAN)", d)
		}
	})

	t.Run("dirty with commit disallowed", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		p := policy.Default()
		p.Commit.Allow = false
		d := PrepareDecide(s, p)
		if d.Status != result.ActionRequired || d.Code != "COMMIT_OR_DISCARD_CHANGES" {
			t.Errorf("got %+v, want ACTION_REQUIRED(COMMIT_OR_DISCARD_CHANGES)", d)
		}
	})

	t.Run("dirty with commit allowed requires confirmation", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		p := policy.Default()
		p.Commit.Allow = true
		d := PrepareDecide(s, p)
		if d.Status != result.HumanApprovalRequired || d.Code != "CLI_CONFIRMATION_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(CLI_CONFIRMATION_REQUIRED)", d)
		}
	})

	t.Run("dirty with commit allowed and confirmed", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		p := policy.Default()
		p.Commit.Allow = true
		p.Approved = true
		d := PrepareDecide(s, p)
		if d.Status != result.Success || d.Code != "COMMIT" {
			t.Errorf("got %+v, want SUCCESS(COMMIT)", d)
		}
	})

	t.Run("dirty with commit allowed and confirmation not required", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		p := policy.Default()
		p.Commit.Allow = true
		p.Commit.RequireConfirmation = false
		d := PrepareDecide(s, p)
		if d.Status != result.Success || d.Code != "COMMIT" {
			t.Errorf("got %+v, want SUCCESS(COMMIT)", d)
		}
	})
}
