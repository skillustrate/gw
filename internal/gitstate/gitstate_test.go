package gitstate

import (
	"context"
	"testing"

	"github.com/gitskill/gw/internal/exec"
)

func TestObserve_NonRepository(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {
			ExitCode: 128,
			Stderr:   "fatal: not a git repository",
		},
	})

	raw, err := Observe(context.Background(), fake, "/not/a/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.IsRepo {
		t.Errorf("expected IsRepo = false, got true")
	}
}

func TestObserve_CleanRepoWithUpstream(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {
			ExitCode: 0,
			Stdout:   "/path/to/repo\n",
		},
		"git rev-parse --git-dir": {
			ExitCode: 0,
			Stdout:   ".git\n",
		},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout: `# branch.oid a1b2c3d4e5
# branch.head feature-auth
# branch.upstream origin/feature-auth
# branch.ab +2 -0
`,
		},
		"git symbolic-ref --short refs/remotes/origin/HEAD": {
			ExitCode: 0,
			Stdout:   "origin/main\n",
		},
	})

	raw, err := Observe(context.Background(), fake, "/path/to/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !raw.IsRepo {
		t.Errorf("expected IsRepo = true")
	}
	if raw.Root != "/path/to/repo" {
		t.Errorf("Root = %q, want '/path/to/repo'", raw.Root)
	}
	if raw.CurrentBranch != "feature-auth" {
		t.Errorf("CurrentBranch = %q, want 'feature-auth'", raw.CurrentBranch)
	}
	if !raw.HasUpstream || raw.Upstream != "origin/feature-auth" {
		t.Errorf("Upstream = %q, HasUpstream = %v", raw.Upstream, raw.HasUpstream)
	}
	if raw.Ahead != 2 || raw.Behind != 0 {
		t.Errorf("Ahead = %d, Behind = %d, want Ahead=2, Behind=0", raw.Ahead, raw.Behind)
	}
	if !raw.Clean {
		t.Errorf("expected Clean = true, got false")
	}
	if raw.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want 'main'", raw.DefaultBranch)
	}
}

func TestObserve_DirtyAndDivergedRepo(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {
			ExitCode: 0,
			Stdout:   "/repo\n",
		},
		"git rev-parse --git-dir": {
			ExitCode: 0,
			Stdout:   ".git\n",
		},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout: `# branch.oid 987654
# branch.head feature-sync
# branch.upstream origin/feature-sync
# branch.ab +3 -2
1 M. N... 100644 100644 100644 abc def file1.go
1 .M N... 100644 100644 100644 abc def file2.go
? untracked.txt
u conflict.go
`,
		},
		"git symbolic-ref --short refs/remotes/origin/HEAD": {
			ExitCode: 1,
		},
		"git rev-parse --verify refs/heads/main": {
			ExitCode: 0,
			Stdout:   "abc\n",
		},
	})

	raw, err := Observe(context.Background(), fake, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !raw.Staged {
		t.Errorf("expected Staged = true")
	}
	if !raw.Unstaged {
		t.Errorf("expected Unstaged = true")
	}
	if !raw.Untracked {
		t.Errorf("expected Untracked = true")
	}
	if !raw.Conflicted {
		t.Errorf("expected Conflicted = true")
	}
	if raw.Clean {
		t.Errorf("expected Clean = false")
	}
	if raw.Ahead != 3 || raw.Behind != 2 {
		t.Errorf("Ahead = %d, Behind = %d", raw.Ahead, raw.Behind)
	}
	if !raw.Diverged {
		t.Errorf("expected Diverged = true")
	}
}

func TestObserve_DetachedHead(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel": {
			ExitCode: 0,
			Stdout:   "/repo\n",
		},
		"git rev-parse --git-dir": {
			ExitCode: 0,
			Stdout:   ".git\n",
		},
		"git status --porcelain=v2 --branch": {
			ExitCode: 0,
			Stdout: `# branch.oid deadbeef
# branch.head (detached)
`,
		},
		"git symbolic-ref --short refs/remotes/origin/HEAD": {
			ExitCode: 1,
		},
		"git rev-parse --verify refs/heads/main": {
			ExitCode: 1,
		},
		"git rev-parse --verify refs/heads/master": {
			ExitCode: 0,
		},
	})

	raw, err := Observe(context.Background(), fake, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !raw.Detached {
		t.Errorf("expected Detached = true")
	}
	if raw.CurrentBranch != "" {
		t.Errorf("expected CurrentBranch = '', got %q", raw.CurrentBranch)
	}
	if raw.DefaultBranch != "master" {
		t.Errorf("DefaultBranch = %q, want 'master'", raw.DefaultBranch)
	}
}