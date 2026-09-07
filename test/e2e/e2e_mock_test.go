package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gitskill/gw/internal/cache"
	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
	"github.com/gitskill/gw/internal/workflow"
)

func boolPtr(b bool) *bool { return &b }

func TestE2E_SimulatedFullLifecycle(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel":     {ExitCode: 0, Stdout: repoDir + "\n"},
		"git rev-parse --git-dir":           {ExitCode: 0, Stdout: repoDir + "/.git\n"},
		"gh auth status":                    {ExitCode: 0, Stdout: "Logged in to github.com\n"},
		"gh repo view --json nameWithOwner": {ExitCode: 0, Stdout: `{"nameWithOwner":"org/repo"}`},
		"gh pr view feature-1 --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 1,
			Stderr:   "no pull requests found for branch \"feature-1\"",
		},
		"git --version": {ExitCode: 0, Stdout: "git version 2.44.0\n"},
		"gh --version":  {ExitCode: 0, Stdout: "gh version 2.45.0 (2024-03-01)\n"},
		"git push --set-upstream origin feature-1": {ExitCode: 0, Stdout: "Everything up-to-date\n"},
	})
	// Before the push, the branch has no upstream. After it, verification
	// re-observes state and expects the upstream tracking branch to exist.
	fake.Sequences["git status --porcelain=v2 --branch"] = []exec.Result{
		{ExitCode: 0, Stdout: "# branch.head feature-1\n# branch.oid 123456\n"},
		{ExitCode: 0, Stdout: "# branch.head feature-1\n# branch.oid 123456\n# branch.upstream origin/feature-1\n# branch.ab +0 -0\n"},
	}

	ctx := context.Background()
	pol := policy.Default()
	pol.Merge.Allow = true
	pol.Approved = true

	params := workflow.Params{
		Dir:           repoDir,
		Runner:        fake,
		Policy:        pol,
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](2 * time.Second),
	}

	// 1. Doctor check
	docEnv := workflow.Doctor(ctx, params)
	if docEnv.Status != result.Success || docEnv.Code != "DOCTOR_OK" {
		t.Fatalf("Doctor check failed: %+v", docEnv)
	}

	// 2. Inspect
	inspectEnv := workflow.Inspect(ctx, params)
	if inspectEnv.Status != result.Success || inspectEnv.Code != "OK" {
		t.Fatalf("Inspect failed: %+v", inspectEnv)
	}

	// 3. PR Ready evaluation
	readyEnv := workflow.PRReady(ctx, params)
	if readyEnv.Status != result.ActionRequired && readyEnv.Code != "PUSH_REQUIRED" && readyEnv.Status != result.Success {
		t.Fatalf("PRReady unexpected result: %+v", readyEnv)
	}

	// 4. Push
	pushEnv := workflow.Push(ctx, params)
	if pushEnv.Status != result.Success && pushEnv.Status != result.Noop {
		t.Fatalf("Push failed: %+v", pushEnv)
	}
}
