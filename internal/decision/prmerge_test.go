package decision

import (
	"testing"

	"github.com/gitskill/gw/internal/ghstate"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
)

func TestDecision_PRMergeDecide(t *testing.T) {
	t.Run("gh unauthenticated", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.Authenticated = false
		d := PRMergeDecide(s, policy.Default())
		if d.Status != result.AuthRequired || d.Code != "GH_UNAUTHENTICATED" {
			t.Errorf("got %+v, want AUTH_REQUIRED(GH_UNAUTHENTICATED)", d)
		}
	})

	t.Run("no pr found", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = nil
		d := PRMergeDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "NO_PR_FOUND" {
			t.Errorf("got %+v, want BLOCKED(NO_PR_FOUND)", d)
		}
	})

	t.Run("already merged", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number: 10,
			State:  "MERGED",
		}
		d := PRMergeDecide(s, policy.Default())
		if d.Status != result.Noop || d.Code != "ALREADY_MERGED" {
			t.Errorf("got %+v, want NOOP(ALREADY_MERGED)", d)
		}
	})

	t.Run("pr closed", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number: 10,
			State:  "CLOSED",
		}
		d := PRMergeDecide(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "PR_CLOSED" {
			t.Errorf("got %+v, want BLOCKED(PR_CLOSED)", d)
		}
	})

	t.Run("merge conflict", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:    10,
			State:     "OPEN",
			Mergeable: boolPtr(false),
		}
		d := PRMergeDecide(s, policy.Default())
		if d.Status != result.Conflict || d.Code != "MERGE_CONFLICT" {
			t.Errorf("got %+v, want CONFLICT(MERGE_CONFLICT)", d)
		}
	})

	t.Run("checks pending", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:       10,
			State:        "OPEN",
			Mergeable:    boolPtr(true),
			ChecksStatus: "pending",
		}
		p := policy.Default()
		p.Merge.RequireChecks = true
		d := PRMergeDecide(s, p)
		if d.Status != result.Wait || d.Code != "CHECKS_PENDING" {
			t.Errorf("got %+v, want WAIT(CHECKS_PENDING)", d)
		}
	})

	t.Run("checks failing", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:       10,
			State:        "OPEN",
			Mergeable:    boolPtr(true),
			ChecksStatus: "failing",
		}
		p := policy.Default()
		p.Merge.RequireChecks = true
		d := PRMergeDecide(s, p)
		if d.Status != result.Blocked || d.Code != "CHECKS_FAILING" {
			t.Errorf("got %+v, want BLOCKED(CHECKS_FAILING)", d)
		}
	})

	t.Run("merge disallowed by policy", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:       10,
			State:        "OPEN",
			Mergeable:    boolPtr(true),
			ChecksStatus: "passing",
		}
		p := policy.Default()
		p.Merge.Allow = false
		d := PRMergeDecide(s, p)
		if d.Status != result.PolicyDenied || d.Code != "MERGE_DISALLOWED" {
			t.Errorf("got %+v, want POLICY_DENIED(MERGE_DISALLOWED)", d)
		}
	})

	t.Run("review approval required", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:         10,
			State:          "OPEN",
			Mergeable:      boolPtr(true),
			ChecksStatus:   "passing",
			ReviewDecision: "REVIEW_REQUIRED",
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Merge.RequireApproval = true
		p.Approved = true
		d := PRMergeDecide(s, p)
		if d.Status != result.HumanApprovalRequired || d.Code != "REVIEW_APPROVAL_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(REVIEW_APPROVAL_REQUIRED)", d)
		}
	})

	t.Run("cli confirmation required (--yes missing)", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:         10,
			State:          "OPEN",
			Mergeable:      boolPtr(true),
			ChecksStatus:   "passing",
			ReviewDecision: "APPROVED",
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Merge.RequireApproval = true
		p.Approved = false // missing --yes
		d := PRMergeDecide(s, p)
		if d.Status != result.HumanApprovalRequired || d.Code != "CLI_CONFIRMATION_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(CLI_CONFIRMATION_REQUIRED)", d)
		}
	})

	t.Run("merge authorized", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:         10,
			State:          "OPEN",
			Mergeable:      boolPtr(true),
			ChecksStatus:   "passing",
			ReviewDecision: "APPROVED",
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Merge.RequireApproval = true
		p.Approved = true
		d := PRMergeDecide(s, p)
		if d.Status != result.Success || d.Code != "MERGE" {
			t.Errorf("got %+v, want SUCCESS(MERGE)", d)
		}
	})
}
