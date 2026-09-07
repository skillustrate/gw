package policy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func boolPtr(b bool) *bool { return &b }
func strPtr(s string) *string { return &s }

func TestPolicy_Defaults(t *testing.T) {
	p := Default()

	if p.Version != 1 {
		t.Errorf("expected Version 1, got %d", p.Version)
	}
	if p.BaseBranch != "main" {
		t.Errorf("expected BaseBranch 'main', got %q", p.BaseBranch)
	}
	if p.Commit.Allow {
		t.Errorf("expected Commit.Allow = false by default")
	}
	if !p.Commit.RequireConfirmation {
		t.Errorf("expected Commit.RequireConfirmation = true by default")
	}
	if !p.Push.Allow || !p.Push.AllowSetUpstream {
		t.Errorf("expected Push.Allow and AllowSetUpstream = true")
	}
	if !p.Push.RequireConfirmation {
		t.Errorf("expected Push.RequireConfirmation = true by default")
	}
	if !p.PullRequest.Create || p.PullRequest.Draft {
		t.Errorf("expected PR Create=true, Draft=false")
	}
	if p.Merge.Allow || !p.Merge.RequireChecks || !p.Merge.RequireApproval {
		t.Errorf("expected Merge Allow=false, RequireChecks=true, RequireApproval=true")
	}
	if p.Sync.Strategy != "ff-only" {
		t.Errorf("expected Sync.Strategy 'ff-only', got %q", p.Sync.Strategy)
	}
	if !p.Sync.RequireConfirmation {
		t.Errorf("expected Sync.RequireConfirmation = true by default")
	}
	if p.Lock.StaleAfter != 10*time.Minute {
		t.Errorf("expected Lock.StaleAfter 10m, got %v", p.Lock.StaleAfter)
	}
}

func TestPolicy_ExplicitFileLoad(t *testing.T) {
	p, err := Load(LoadOptions{
		ExplicitConfigPath: filepath.Join("testdata", "custom.yaml"),
	})
	if err != nil {
		t.Fatalf("unexpected error loading valid custom.yaml: %v", err)
	}

	if p.BaseBranch != "develop" {
		t.Errorf("BaseBranch = %q, want 'develop'", p.BaseBranch)
	}
	if !p.Commit.Allow {
		t.Errorf("Commit.Allow = %v, want true", p.Commit.Allow)
	}
	if p.Push.Allow {
		t.Errorf("Push.Allow = %v, want false", p.Push.Allow)
	}
	if p.Lock.StaleAfter != 5*time.Minute {
		t.Errorf("Lock.StaleAfter = %v, want 5m", p.Lock.StaleAfter)
	}
}

func TestPolicy_UntrustedRepoConfigIgnoredByDefault(t *testing.T) {
	tempDir := t.TempDir()
	maliciousConfig := filepath.Join(tempDir, ".gw.yml")
	err := os.WriteFile(maliciousConfig, []byte("version: 1\nbase_branch: main\nmerge:\n  allow: true\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp .gw.yml: %v", err)
	}

	// Case 1: AllowRepoConfig is false -> repo file must have NO effect
	p1, err := Load(LoadOptions{
		AllowRepoConfig: false,
		RepoRoot:        tempDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p1.Merge.Allow {
		t.Errorf("SECURITY INVARIANT VIOLATED: untrusted .gw.yml enabled merge.allow without --allow-repo-config")
	}

	// Case 2: AllowRepoConfig is true -> repo file is honored
	p2, err := Load(LoadOptions{
		AllowRepoConfig: true,
		RepoRoot:        tempDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p2.Merge.Allow {
		t.Errorf("expected merge.allow = true when AllowRepoConfig is explicitly enabled")
	}
}

func TestPolicy_PrecedenceLayers(t *testing.T) {
	// Defaults < Config File < Env < Flags
	p, err := Load(LoadOptions{
		ExplicitConfigPath: filepath.Join("testdata", "custom.yaml"), // sets BaseBranch: develop, Commit.Allow: true
		Env: map[string]string{
			"GW_BASE_BRANCH": "staging",
		},
		Flags: &PolicyOverrides{
			BaseBranch:  strPtr("release-1.0"),
			CommitAllow: boolPtr(false),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CLI flag should win over both env and file
	if p.BaseBranch != "release-1.0" {
		t.Errorf("BaseBranch = %q, want 'release-1.0' (flag precedence)", p.BaseBranch)
	}
	if p.Commit.Allow {
		t.Errorf("Commit.Allow = %v, want false (flag precedence)", p.Commit.Allow)
	}
}

func TestPolicy_ValidationErrors(t *testing.T) {
	// Unknown key
	_, err := Load(LoadOptions{
		ExplicitConfigPath: filepath.Join("testdata", "unknown_key.yaml"),
	})
	if err == nil {
		t.Errorf("expected error on unknown key, got nil")
	}

	// Invalid strategy
	_, err = Load(LoadOptions{
		ExplicitConfigPath: filepath.Join("testdata", "invalid_strategy.yaml"),
	})
	if err == nil {
		t.Errorf("expected error on invalid sync.strategy, got nil")
	}

	// Invalid merge method
	_, err = Load(LoadOptions{
		Flags: &PolicyOverrides{
			MergeMethod: strPtr("invalid-method"),
		},
	})
	if err == nil {
		t.Errorf("expected error on invalid merge.method, got nil")
	}
}

func TestPolicy_MergeMethod(t *testing.T) {
	p := Default()
	if p.Merge.Method != "squash" {
		t.Errorf("expected default Merge.Method = 'squash', got %q", p.Merge.Method)
	}

	// Override via env
	p2, err := Load(LoadOptions{
		Env: map[string]string{
			"GW_MERGE_METHOD": "rebase",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p2.Merge.Method != "rebase" {
		t.Errorf("expected Merge.Method = 'rebase', got %q", p2.Merge.Method)
	}

	// Override via flag
	p3, err := Load(LoadOptions{
		Env: map[string]string{
			"GW_MERGE_METHOD": "rebase",
		},
		Flags: &PolicyOverrides{
			MergeMethod: strPtr("merge"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p3.Merge.Method != "merge" {
		t.Errorf("expected Merge.Method = 'merge', got %q", p3.Merge.Method)
	}
}