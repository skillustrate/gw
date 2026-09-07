package decision

import (
	"testing"

	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
)

func TestDecision_PushDecide(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		s := baseRepoState()
		s.Repository.IsGitRepo = false
		d := PushDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "NOT_A_REPO" {
			t.Errorf("got %+v, want BLOCKED(NOT_A_REPO)", d)
		}
	})

	t.Run("push disallowed by policy", func(t *testing.T) {
		s := baseRepoState()
		p := policy.Default()
		p.Push.Allow = false
		d := PushDecide(s, p)
		if d.Status != result.PolicyDenied || d.Code != "PUSH_DISALLOWED" {
			t.Errorf("got %+v, want POLICY_DENIED(PUSH_DISALLOWED)", d)
		}
	})

	t.Run("branch diverged", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = true
		s.Branch.Diverged = true
		d := PushDecide(s, policy.Default())
		if d.Status != result.Conflict || d.Code != "BRANCH_DIVERGED" {
			t.Errorf("got %+v, want CONFLICT(BRANCH_DIVERGED)", d)
		}
	})

	t.Run("already pushed (ahead == 0)", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = true
		s.Branch.Ahead = 0
		d := PushDecide(s, policy.Default())
		if d.Status != result.Noop || d.Code != "ALREADY_PUSHED" {
			t.Errorf("got %+v, want NOOP(ALREADY_PUSHED)", d)
		}
	})

	t.Run("ready to push (ahead > 0) requires confirmation", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = true
		s.Branch.Ahead = 2
		d := PushDecide(s, policy.Default())
		if d.Status != result.HumanApprovalRequired || d.Code != "CLI_CONFIRMATION_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(CLI_CONFIRMATION_REQUIRED)", d)
		}
	})

	t.Run("ready to push (ahead > 0) with confirmation given", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = true
		s.Branch.Ahead = 2
		p := policy.Default()
		p.Approved = true
		d := PushDecide(s, p)
		if d.Status != result.Success || d.Code != "PUSH" {
			t.Errorf("got %+v, want SUCCESS(PUSH)", d)
		}
	})

	t.Run("ready to push (ahead > 0) with confirmation not required", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = true
		s.Branch.Ahead = 2
		p := policy.Default()
		p.Push.RequireConfirmation = false
		d := PushDecide(s, p)
		if d.Status != result.Success || d.Code != "PUSH" {
			t.Errorf("got %+v, want SUCCESS(PUSH)", d)
		}
	})

	t.Run("no upstream with set-upstream disallowed", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = false
		p := policy.Default()
		p.Push.AllowSetUpstream = false
		d := PushDecide(s, p)
		if d.Status != result.PolicyDenied || d.Code != "SET_UPSTREAM_DISALLOWED" {
			t.Errorf("got %+v, want POLICY_DENIED(SET_UPSTREAM_DISALLOWED)", d)
		}
	})

	t.Run("no upstream with set-upstream allowed requires confirmation", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = false
		p := policy.Default()
		p.Push.AllowSetUpstream = true
		d := PushDecide(s, p)
		if d.Status != result.HumanApprovalRequired || d.Code != "CLI_CONFIRMATION_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(CLI_CONFIRMATION_REQUIRED)", d)
		}
	})

	t.Run("no upstream with set-upstream allowed and confirmed", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = false
		p := policy.Default()
		p.Push.AllowSetUpstream = true
		p.Approved = true
		d := PushDecide(s, p)
		if d.Status != result.Success || d.Code != "PUSH_WITH_TRACKING" {
			t.Errorf("got %+v, want SUCCESS(PUSH_WITH_TRACKING)", d)
		}
	})
}
