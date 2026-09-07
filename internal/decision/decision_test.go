package decision

import (
	"testing"
	"time"

	"github.com/gitskill/gw/internal/ghstate"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

func boolPtr(b bool) *bool { return &b }

func baseRepoState() state.RepoState {
	return state.RepoState{
		SchemaVersion: 1,
		ObservedAt:    time.Now().UTC(),
		EngineVersion: "0.1.0",
		Repository: state.RepositoryState{
			IsGitRepo:     true,
			Root:          "/repo",
			DefaultBranch: "main",
			CurrentBranch: "feature/test",
			Detached:      false,
		},
		WorkingTree: state.WorkingTreeState{
			Clean: true,
		},
		Branch: state.BranchState{
			IsDefault:   false,
			HasUpstream: true,
			Upstream:    "origin/feature/test",
			Ahead:       0,
			Behind:      0,
			Diverged:    false,
		},
		GitHub: state.GitHubState{
			Available:     true,
			Authenticated: true,
			Repository:    "org/repo",
			PullRequest:   nil,
		},
	}
}

func TestDecision_MasterPromptMinimumSuite(t *testing.T) {
	// 1. clean+pushed+no PR -> CREATE_PR (PRCreate)
	t.Run("clean+pushed+no PR -> CREATE_PR", func(t *testing.T) {
		s := baseRepoState()
		p := policy.Default()
		d := PRCreate(s, p)
		if d.Status != result.Success || d.Code != "CREATE_PR" {
			t.Errorf("got %+v, want SUCCESS(CREATE_PR)", d)
		}
	})

	// 2. clean+pushed+existing PR -> NOOP (PRCreate)
	t.Run("clean+pushed+existing PR -> NOOP", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number: 42,
			State:  "OPEN",
		}
		p := policy.Default()
		d := PRCreate(s, p)
		if d.Status != result.Noop || d.Code != "PR_ALREADY_EXISTS" {
			t.Errorf("got %+v, want NOOP(PR_ALREADY_EXISTS)", d)
		}
	})

	// 3. dirty -> BLOCKED (PRCreate)
	t.Run("dirty -> BLOCKED", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		s.WorkingTree.Unstaged = true
		p := policy.Default()
		d := PRCreate(s, p)
		if d.Status != result.Blocked || d.Code != "WORKTREE_DIRTY" {
			t.Errorf("got %+v, want BLOCKED(WORKTREE_DIRTY)", d)
		}
	})

	// 4. unpushed+no PR -> PUSH / PUSH_REQUIRED
	t.Run("unpushed+no PR -> PUSH_REQUIRED", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Ahead = 1
		p := policy.Default()
		d := PRCreate(s, p)
		if d.Status != result.ActionRequired || d.Code != "PUSH_REQUIRED" {
			t.Errorf("got %+v, want ACTION_REQUIRED(PUSH_REQUIRED)", d)
		}
	})

	// 5. no upstream -> PUSH_WITH_TRACKING (Push)
	t.Run("no upstream -> PUSH_WITH_TRACKING", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.HasUpstream = false
		s.Branch.Upstream = ""
		p := policy.Default()
		p.Approved = true
		d := Push(s, p)
		if d.Status != result.Success || d.Code != "PUSH_WITH_TRACKING" {
			t.Errorf("got %+v, want SUCCESS(PUSH_WITH_TRACKING)", d)
		}
	})

	// 6. behind base -> SYNC_FF_ONLY / BLOCKED if dirty
	t.Run("behind base -> SYNC_FF_ONLY", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Behind = 3
		p := policy.Default()
		p.Approved = true
		d := Sync(s, p)
		if d.Status != result.Success || d.Code != "SYNC_FF_ONLY" {
			t.Errorf("got %+v, want SUCCESS(SYNC_FF_ONLY)", d)
		}
	})

	// 7. diverged -> CONFLICT
	t.Run("diverged -> CONFLICT", func(t *testing.T) {
		s := baseRepoState()
		s.Branch.Ahead = 2
		s.Branch.Behind = 2
		s.Branch.Diverged = true
		p := policy.Default()
		d := Sync(s, p)
		if d.Status != result.Conflict || d.Code != "BRANCH_DIVERGED" {
			t.Errorf("got %+v, want CONFLICT(BRANCH_DIVERGED)", d)
		}
	})

	// 8. failing checks -> BLOCKED (PRMerge)
	t.Run("failing checks -> BLOCKED", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:         10,
			State:          "OPEN",
			ChecksStatus:   "failing",
			ReviewDecision: "APPROVED",
			Mergeable:      boolPtr(true),
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Approved = true
		d := PRMerge(s, p)
		if d.Status != result.Blocked || d.Code != "CHECKS_FAILING" {
			t.Errorf("got %+v, want BLOCKED(CHECKS_FAILING)", d)
		}
	})

	// 9. merge conflict -> CONFLICT (PRMerge)
	t.Run("merge conflict -> CONFLICT", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:         10,
			State:          "OPEN",
			Mergeable:      boolPtr(false),
			ChecksStatus:   "passing",
			ReviewDecision: "APPROVED",
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Approved = true
		d := PRMerge(s, p)
		if d.Status != result.Conflict || d.Code != "MERGE_CONFLICT" {
			t.Errorf("got %+v, want CONFLICT(MERGE_CONFLICT)", d)
		}
	})

	// 10. already merged -> NOOP (PRMerge)
	t.Run("already merged -> NOOP", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number: 10,
			State:  "MERGED",
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Approved = true
		d := PRMerge(s, p)
		if d.Status != result.Noop || d.Code != "ALREADY_MERGED" {
			t.Errorf("got %+v, want NOOP(ALREADY_MERGED)", d)
		}
	})

	// 11. gh unauthenticated -> AUTH_REQUIRED
	t.Run("gh unauthenticated -> AUTH_REQUIRED", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.Authenticated = false
		p := policy.Default()
		d := PRCreate(s, p)
		if d.Status != result.AuthRequired || d.Code != "GH_UNAUTHENTICATED" {
			t.Errorf("got %+v, want AUTH_REQUIRED(GH_UNAUTHENTICATED)", d)
		}
	})

	// 12. two-factor approval verification for merge
	t.Run("two-factor approval: --yes alone is not enough without review approval", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:         12,
			State:          "OPEN",
			ChecksStatus:   "passing",
			ReviewDecision: "REVIEW_REQUIRED", // Not approved
			Mergeable:      boolPtr(true),
		}
		p := policy.Default()
		p.Merge.Allow = true
		p.Approved = true // --yes passed
		d := PRMerge(s, p)
		if d.Status != result.HumanApprovalRequired || d.Code != "REVIEW_APPROVAL_REQUIRED" {
			t.Errorf("got %+v, want HUMAN_APPROVAL_REQUIRED(REVIEW_APPROVAL_REQUIRED)", d)
		}
	})
}

func TestDecision_Inspect(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		s := baseRepoState()
		s.Repository.IsGitRepo = false
		d := Inspect(s, policy.Default())
		// Inspect is a passive, read-only report: it never blocks, it just
		// observes and reports what it found (consistent with NOT_A_REPO
		// used as a plain observation code elsewhere, e.g. PushDecide).
		if d.Status != result.Success || d.Code != "NOT_A_REPO" {
			t.Errorf("got %+v, want SUCCESS(NOT_A_REPO)", d)
		}
	})

	t.Run("detached HEAD", func(t *testing.T) {
		s := baseRepoState()
		s.Repository.Detached = true
		d := Inspect(s, policy.Default())
		// Inspect does not gate on detached HEAD; the detached flag is
		// reported via the observed RepoState payload, not a blocking code.
		if d.Status != result.Success || d.Code != "OK" {
			t.Errorf("got %+v, want SUCCESS(OK)", d)
		}
	})

	t.Run("healthy repo", func(t *testing.T) {
		s := baseRepoState()
		d := Inspect(s, policy.Default())
		if d.Status != result.Success || d.Code != "OK" {
			t.Errorf("got %+v, want SUCCESS(OK)", d)
		}
	})
}

func TestDecision_Prepare(t *testing.T) {
	t.Run("clean working tree", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = true
		d := Prepare(s, policy.Default())
		if d.Status != result.Noop || d.Code != "CLEAN" {
			t.Errorf("got %+v, want NOOP(CLEAN)", d)
		}
	})

	t.Run("dirty working tree with commit disallowed", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		p := policy.Default()
		p.Commit.Allow = false
		d := Prepare(s, p)
		if d.Status != result.ActionRequired || d.Code != "COMMIT_OR_DISCARD_CHANGES" {
			t.Errorf("got %+v, want ACTION_REQUIRED(COMMIT_OR_DISCARD_CHANGES)", d)
		}
	})

	t.Run("dirty working tree with commit allowed", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		p := policy.Default()
		p.Commit.Allow = true
		p.Approved = true
		d := Prepare(s, p)
		if d.Status != result.Success || d.Code != "COMMIT" {
			t.Errorf("got %+v, want SUCCESS(COMMIT)", d)
		}
	})
}

func TestDecision_PRReady(t *testing.T) {
	t.Run("ready for PR", func(t *testing.T) {
		s := baseRepoState()
		d := PRReady(s, policy.Default())
		// PRReady uses the same CREATE_PR code as PRCreate for the
		// ready-to-create-a-PR outcome (consistent action-oriented naming).
		if d.Status != result.Success || d.Code != "CREATE_PR" {
			t.Errorf("got %+v, want SUCCESS(CREATE_PR)", d)
		}
	})

	t.Run("dirty tree not ready", func(t *testing.T) {
		s := baseRepoState()
		s.WorkingTree.Clean = false
		d := PRReady(s, policy.Default())
		if d.Status != result.Blocked || d.Code != "WORKTREE_DIRTY" {
			t.Errorf("got %+v, want BLOCKED(WORKTREE_DIRTY)", d)
		}
	})
}

func TestDecision_PRStatus(t *testing.T) {
	t.Run("no PR found", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = nil
		d := PRStatus(s, policy.Default())
		if d.Status != result.Noop || d.Code != "NO_PR_FOUND" {
			t.Errorf("got %+v, want NOOP(NO_PR_FOUND)", d)
		}
	})

	t.Run("open PR with passing checks", func(t *testing.T) {
		s := baseRepoState()
		s.GitHub.PullRequest = &ghstate.PullRequest{
			Number:       15,
			State:        "OPEN",
			ChecksStatus: "passing",
		}
		d := PRStatus(s, policy.Default())
		// PRStatus reports a single observed-successfully code; the granular
		// open/passing/failing detail lives in the returned PullRequest data,
		// not in a per-state code.
		if d.Status != result.Success || d.Code != "PR_STATUS_OBSERVED" {
			t.Errorf("got %+v, want SUCCESS(PR_STATUS_OBSERVED)", d)
		}
	})
}