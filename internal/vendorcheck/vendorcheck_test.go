package vendorcheck

import (
	"os"
	"path/filepath"
	"testing"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root containing go.mod")
		}
		dir = parent
	}
}

func TestVendorCheck_NoVendorImports(t *testing.T) {
	root := findRepoRoot(t)

	for _, sub := range []string{"internal", "cmd"} {
		dir := filepath.Join(root, sub)
		violations, err := CheckNoVendorImports(dir)
		if err != nil {
			t.Fatalf("unexpected error scanning %s/: %v", sub, err)
		}

		if len(violations) > 0 {
			t.Errorf("found forbidden vendor imports in %s/:\n%v", sub, violations)
		}
	}
}