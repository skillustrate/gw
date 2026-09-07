package integration

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

// newDisposableRepo sets up a local repository connected to a local bare remote repository.
func newDisposableRepo(t *testing.T) (string, string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found, skipping integration test")
	}

	tempDir := t.TempDir()
	remoteDir := filepath.Join(tempDir, "remote.git")
	workDir := filepath.Join(tempDir, "work")

	ctx := context.Background()

	// Initialize bare remote repo
	runGit(t, ctx, tempDir, "init", "--bare", remoteDir)

	// Initialize working repo
	runGit(t, ctx, tempDir, "init", workDir)

	// Configure test user identity
	runGit(t, ctx, workDir, "config", "user.name", "Test User")
	runGit(t, ctx, workDir, "config", "user.email", "test@example.com")
	runGit(t, ctx, workDir, "config", "commit.gpgsign", "false")

	// Add remote
	runGit(t, ctx, workDir, "remote", "add", "origin", remoteDir)

	return workDir, remoteDir
}

func runGit(t *testing.T, ctx context.Context, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput:\n%s", args, dir, err, string(out))
	}
	return string(out)
}