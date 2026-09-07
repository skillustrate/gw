package decision

import (
	"testing"

	"github.com/gitskill/gw/internal/ghstate"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
)

func TestDecision_PRCreateDecide(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		s := baseRepoState()
		s.Repository.IsGitRepo = false
		d := PRCreateDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "NOT_A_REPO" {
			t.Errorf("got %+v, want BLOCKED(NOT_A_REPO)", d)
		}
	})

	t.Run("on default branch", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.IsDefault = true
		d := PRCreateDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "ON_DEFAULT_BRANCH" {
			t.Errorf("got %+v, want BLOCKED(ON_DEFAULT_BRANCH)", d)
		}
	})

	t.Run("existing open PR", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number: 42,
			State:  "OPEN",
		}
		d := PRCreateDecide(s, policy.Default())
		if d.Status != result.Noop || d.Code != "PR_ALREADY_EXISTS" {
			t.Errorf("got %+v, want NOOP(PR_ALREADY_EXISTS)", d)
		}
	})

	t.Run("dirty working tree", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		d := PRCreateDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "WORKTREE_DIRTY" {
			t.Errorf("got %+v, want BLOCKED(WORKTREE_DIRTY)", d)
		}
	})

	t.Run("unpushed and push disallowed", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Ahead = 1
		p := policy.Default()
		p.Push.Allow = false
		d := PRCreateDecide(s, p)
		if d.Status != result.PolicyDenied || d.Code != "PUSH_DISALLOWED" {
			t.Errorf("got %+v, want POLICY_DENIED(PUSH_DISALLOWED)", d)
		}
	})

	t.Run("unpushed and push allowed", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Ahead = 1
		p := policy.Default()
		p.Push.Allow = true
		d := PRCreateDecide(s, p)
		if d.Status != result.ActionRequired || d.Code != "PUSH_REQUIRED" {
			t.Errorf("got %+v, want ACTION_REQUIRED(PUSH_REQUIRED)", d)
		}
	})

	t.Run("pr create disallowed by policy", func(t *testing.T) {
		s := baseRepoState()
		p := policy.Default()
		p.PullRequest.Create = false
		d := PRCreateDecide(s, p)
		if d.Status != result.PolicyDenied || d.Code != "PR_CREATE_DISALLOWED" {
			t.Errorf("got %+v, want POLICY_DENIED(PR_CREATE_DISALLOWED)", d)
		}
	})

	t.Run("gh unauthenticated", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.Authenticated = false
		d := PRCreateDecide(s, policy.Default())
		if d.Status != result.AuthRequired || d.Code != "GH_UNAUTHENTICATED" {
			t.Errorf("got %+v, want AUTH_REQUIRED(GH_UNAUTHENTICATED)", d)
		}
	})

	t.Run("ready to create PR", func(t *testing.T) {
		s := baseRepoState()
		d := PRCreateDecide(s, policy.Default())
		if d.Status != result.Success || d.Code != "CREATE_PR" {
			t.Errorf("got %+v, want SUCCESS(CREATE_PR)", d)
		}
	})
}
