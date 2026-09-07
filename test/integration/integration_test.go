package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitskill/gw/internal/cache"
	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
	"github.com/gitskill/gw/internal/workflow"
)

func TestIntegration_PreparePushSyncRoundtrip(t *testing.T) {
	workDir, _ := newDisposableRepo(t)
	runner := exec.RealRunner{}
	ctx := context.Background()

	// 1. Initial inspect (empty repo)
	p := workflow.Params{
		Dir:           workDir,
		Runner:        runner,
		Policy:        policy.Default(),
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	inspectEnv := workflow.Inspect(ctx, p)
	if inspectEnv.Status != result.Success {
		t.Fatalf("inspect failed: %+v", inspectEnv)
	}

	// 2. Create a test file
	testFile := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 3. Prepare with commit.allow: true
	p.Policy.Commit.Allow = true
	p.Policy.Approved = true
	p.CommitMessage = "Initial commit"
	prepareEnv := workflow.Prepare(ctx, p)
	if prepareEnv.Status != result.Success || prepareEnv.Code != "COMMITTED" {
		t.Fatalf("prepare failed: %+v", prepareEnv)
	}

	// 4. Push to remote
	pushEnv := workflow.Push(ctx, p)
	if pushEnv.Status != result.Success || pushEnv.Code != "PUSHED" {
		t.Fatalf("push failed: %+v", pushEnv)
	}

	// 5. Verify push is now NOOP
	pushNoopEnv := workflow.Push(ctx, p)
	if pushNoopEnv.Status != result.Noop || pushNoopEnv.Code != "ALREADY_PUSHED" {
		t.Fatalf("expected NOOP(ALREADY_PUSHED), got: %+v", pushNoopEnv)
	}

	// 6. Sync (already up to date -> NOOP)
	syncEnv := workflow.Sync(ctx, p)
	if syncEnv.Status != result.Noop || syncEnv.Code != "ALREADY_SYNCHRONIZED" {
		t.Fatalf("expected NOOP(ALREADY_SYNCHRONIZED), got: %+v", syncEnv)
	}
}

func TestIntegration_PrepareDisallowedStopsOnDirty(t *testing.T) {
	workDir, _ := newDisposableRepo(t)
	runner := exec.RealRunner{}
	ctx := context.Background()

	testFile := filepath.Join(workDir, "dirty.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted content\n"), 0644); err != nil {
		t.Fatalf("failed to write dirty file: %v", err)
	}

	p := workflow.Params{
		Dir:           workDir,
		Runner:        runner,
		Policy:        policy.Default(), // Commit.Allow is false by default
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	env := workflow.Prepare(ctx, p)
	if env.Status != result.ActionRequired || env.Code != "COMMIT_OR_DISCARD_CHANGES" {
		t.Fatalf("expected ACTION_REQUIRED(COMMIT_OR_DISCARD_CHANGES), got: %+v", env)
	}
}