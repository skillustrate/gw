package exec

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestFakeRunner_DeterministicLookup(t *testing.T) {
	fake := NewFakeRunner(map[string]Result{
		"git status --porcelain=v2": {
			ExitCode: 0,
			Stdout:   "# branch.oid abc1234\n",
		},
		"gh pr view --json number": {
			ExitCode: 1,
			Stderr:   "no pull requests found\n",
		},
	})

	ctx := context.Background()

	// Call 1
	res, err := fake.Run(ctx, "/repo", "git", "status", "--porcelain=v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "branch.oid") {
		t.Errorf("unexpected result: %+v", res)
	}

	// Call 2
	res, err = fake.Run(ctx, "/repo", "gh", "pr", "view", "--json", "number")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 1 || !strings.Contains(res.Stderr, "no pull requests") {
		t.Errorf("unexpected result: %+v", res)
	}

	// Verify call tracking
	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(fake.Calls))
	}
	if fake.Calls[0].Key() != "git status --porcelain=v2" {
		t.Errorf("expected first call key 'git status --porcelain=v2', got %q", fake.Calls[0].Key())
	}
}

func TestFakeRunner_UnmappedFailsLoudly(t *testing.T) {
	fake := NewFakeRunner(nil)
	ctx := context.Background()

	_, err := fake.Run(ctx, "/repo", "git", "push", "origin", "main")
	if err == nil {
		t.Fatal("expected error on unmapped invocation, got nil")
	}
	if !strings.Contains(err.Error(), "unmapped invocation") {
		t.Errorf("expected unmapped invocation error message, got %v", err)
	}
}

func TestRealRunner_ExecuteGitVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH, skipping RealRunner integration check")
	}

	runner := RealRunner{Timeout: 5 * time.Second}
	ctx := context.Background()

	res, err := runner.Run(ctx, "", "git", "--version")
	if err != nil {
		t.Fatalf("unexpected error running git --version: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
	if !strings.HasPrefix(strings.TrimSpace(res.Stdout), "git version") {
		t.Errorf("expected git version output, got %q", res.Stdout)
	}
}