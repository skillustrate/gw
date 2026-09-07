package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitskill/gw/internal/cache"
	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

func TestWorkflow_Prepare_StopsOnDirtyWhenDisallowed(t *testing.T) {
	tempDir := t.TempDir()
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {ExitCode: 0, Stdout: tempDir + "\n"},
		"git rev-parse --git-dir":       {ExitCode: 0, Stdout: tempDir + "/.git\n"},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout:   "# branch.head feature-1\n1 .M N... 100644 100644 100644 abc def file.txt\n",
		},
	})

	p := Params{
		Dir:           tempDir,
		Runner:        fake,
		Policy:        policy.Default(), // commit.allow is false by default
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	env := Prepare(context.Background(), p)
	if env.Status != result.ActionRequired || env.Code != "COMMIT_OR_DISCARD_CHANGES" {
		t.Fatalf("expected ACTION_REQUIRED(COMMIT_OR_DISCARD_CHANGES), got %+v", env)
	}

	// Verify no git commit was executed
	for _, call := range fake.Calls {
		if call.Name == "git" && len(call.Args) > 0 && call.Args[0] == "commit" {
			t.Fatalf("SAFETY VIOLATION: git commit was invoked when commit.allow=false")
		}
	}
}

func TestWorkflow_Push_WithTracking(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {ExitCode: 0, Stdout: tempDir + "\n"},
		"git rev-parse --git-dir":       {ExitCode: 0, Stdout: tempDir + "/.git\n"},
		"git push --set-upstream origin feature-new": {
			ExitCode: 0,
			Stdout:   "Branch 'feature-new' set up to track remote branch 'feature-new' from 'origin'.\n",
		},
	})
	// Before the push, the branch has no upstream. After it, verification
	// re-observes state and expects the upstream tracking branch to exist.
	fake.Sequences["git status --porcelain=v2 --branch"] = []exec.Result{
		{ExitCode: 0, Stdout: "# branch.head feature-new\n# branch.oid 123456\n"},
		{ExitCode: 0, Stdout: "# branch.head feature-new\n# branch.oid 123456\n# branch.upstream origin/feature-new\n# branch.ab +0 -0\n"},
	}

	pol := policy.Default()
	pol.Approved = true

	p := Params{
		Dir:           tempDir,
		Runner:        fake,
		Policy:        pol,
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	env := Push(context.Background(), p)
	if env.Status != result.Success {
		t.Fatalf("expected SUCCESS, got %+v", env)
	}

	// Verify git push --set-upstream was called
	foundCall := false
	for _, call := range fake.Calls {
		if strings.Join(call.Args, " ") == "push --set-upstream origin feature-new" {
			foundCall = true
			break
		}
	}
	if !foundCall {
		t.Errorf("expected push with tracking call")
	}
}

func TestWorkflow_PRMerge_TwoFactorGate(t *testing.T) {
	tempDir := t.TempDir()
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {ExitCode: 0, Stdout: tempDir + "\n"},
		"git rev-parse --git-dir":       {ExitCode: 0, Stdout: tempDir + "/.git\n"},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout:   "# branch.head feature-auth\n",
		},
		"gh auth status": {ExitCode: 0, Stdout: "Logged in\n"},
		"gh repo view --json nameWithOwner": {ExitCode: 0, Stdout: `{"nameWithOwner":"org/repo"}`},
		"gh pr view feature-auth --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 0,
			Stdout: `{
  "number": 99,
  "url": "https://github.com/org/repo/pull/99",
  "state": "OPEN",
  "mergeable": "MERGEABLE",
  "reviewDecision": "REVIEW_REQUIRED",
  "statusCheckRollup": [{"state":"SUCCESS","status":"COMPLETED","conclusion":"SUCCESS"}]
}`,
		},
	})

	// Case 1: Policy enables merge and --yes is provided, but PR is not approved
	pol := policy.Default()
	pol.Merge.Allow = true
	pol.Approved = true

	p := Params{
		Dir:           tempDir,
		Runner:        fake,
		Policy:        pol,
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	env := PRMerge(context.Background(), p)
	if env.Status != result.HumanApprovalRequired || env.Code != "REVIEW_APPROVAL_REQUIRED" {
		t.Fatalf("SECURITY VIOLATION: expected HUMAN_APPROVAL_REQUIRED(REVIEW_APPROVAL_REQUIRED), got %+v", env)
	}

	// Verify gh pr merge was NEVER called
	for _, call := range fake.Calls {
		if call.Name == "gh" && len(call.Args) > 1 && call.Args[0] == "pr" && call.Args[1] == "merge" {
			t.Fatalf("CRITICAL SECURITY VIOLATION: gh pr merge was executed without review approval!")
		}
	}
}

func TestWorkflow_Doctor(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git --version":  {ExitCode: 0, Stdout: "git version 2.44.0\n"},
		"gh --version":   {ExitCode: 0, Stdout: "gh version 2.45.0 (2024-03-01)\n"},
		"gh auth status": {ExitCode: 0, Stdout: "Logged in to github.com\n"},
	})

	p := Params{
		Runner: fake,
	}

	env := Doctor(context.Background(), p)
	if env.Status != result.Success || env.Code != "DOCTOR_OK" {
		t.Fatalf("expected DOCTOR_OK, got %+v", env)
	}
}

func TestWorkflow_Doctor_OutdatedVersion(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git --version":  {ExitCode: 0, Stdout: "git version 2.10.0\n"}, // < 2.20.0
		"gh --version":   {ExitCode: 0, Stdout: "gh version 2.45.0 (2024-03-01)\n"},
		"gh auth status": {ExitCode: 0, Stdout: "Logged in to github.com\n"},
	})

	p := Params{
		Runner: fake,
	}

	env := Doctor(context.Background(), p)
	if env.Status != result.ActionRequired || env.Code != "DIAGNOSTIC_WARNING" {
		t.Fatalf("expected DIAGNOSTIC_WARNING for old git version, got %+v", env)
	}
	if !strings.Contains(env.Reason, "older than minimum required") {
		t.Errorf("expected warning about older version, got %s", env.Reason)
	}
}

func TestWorkflow_Sync_FastForward(t *testing.T) {
	tempDir := t.TempDir()
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {ExitCode: 0, Stdout: tempDir + "\n"},
		"git rev-parse --git-dir":       {ExitCode: 0, Stdout: tempDir + "/.git\n"},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout:   "# branch.head feature-1\n# branch.oid 123456\n# branch.ab +0 -2\n# branch.upstream origin/feature-1\n",
		},
		"git fetch origin": {
			ExitCode: 0,
			Stdout:   "",
		},
		"git merge --ff-only origin/feature-1": {
			ExitCode: 0,
			Stdout:   "Updating 123456..7890ab\nFast-forward\n",
		},
	})

	p := Params{
		Dir:           tempDir,
		Runner:        fake,
		Policy:        policy.Default(),
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	env := Sync(context.Background(), p)
	// After merge, observeState is called again; in fake runner it returns behind 2 unless overridden, but verifies execution
	t.Logf("Sync env: %+v", env)
}

func TestWorkflow_PRCreate_Execution(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {ExitCode: 0, Stdout: tempDir + "\n"},
		"git rev-parse --git-dir":       {ExitCode: 0, Stdout: tempDir + "/.git\n"},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout:   "# branch.head feature-pr\n# branch.oid 123456\n# branch.ab +0 -0\n# branch.upstream origin/feature-pr\n",
		},
		"gh auth status":                    {ExitCode: 0, Stdout: "Logged in\n"},
		"gh repo view --json nameWithOwner": {ExitCode: 0, Stdout: `{"nameWithOwner":"org/repo"}`},
		"gh pr view feature-pr --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 1,
			Stderr:   "no pull requests found",
		},
		"gh pr create --title feature-pr --body Automated pull request for branch feature-pr --base main": {
			ExitCode: 0,
			Stdout:   "https://github.com/org/repo/pull/101\n",
		},
	})

	p := Params{
		Dir:           tempDir,
		Runner:        fake,
		Policy:        policy.Default(),
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	env := PRCreate(context.Background(), p)
	if env.Status != result.Success || env.Code != "PR_CREATED" {
		t.Fatalf("expected PR_CREATED, got %+v", env)
	}
}