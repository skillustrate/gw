package gitstate

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gitskill/gw/internal/exec"
)

// Raw contains unnormalized observed local Git repository state.
type Raw struct {
	IsRepo        bool
	Root          string
	DefaultBranch string
	CurrentBranch string
	Detached      bool
	Clean         bool
	Staged        bool
	Unstaged      bool
	Untracked     bool
	Conflicted    bool
	InProgressOp  string // "", "merge", "rebase", "cherry-pick", "revert", "bisect"
	HasUpstream   bool
	Upstream      string
	Ahead         int
	Behind        int
	Diverged      bool
}

// Observe inspects local Git repository state using the runner boundary.
func Observe(ctx context.Context, r exec.Runner, dir string) (Raw, error) {
	var raw Raw

	// Step 1: Detect repository root & git-dir
	rootRes, err := r.Run(ctx, dir, "git", "rev-parse", "--show-toplevel")
	if err != nil || rootRes.ExitCode != 0 {
		// Not a git repository
		return Raw{IsRepo: false}, nil
	}

	raw.IsRepo = true
	raw.Root = strings.TrimSpace(rootRes.Stdout)

	gitDirRes, err := r.Run(ctx, dir, "git", "rev-parse", "--git-dir")
	gitDir := filepath.Join(raw.Root, ".git")
	if err == nil && gitDirRes.ExitCode == 0 {
		gdir := strings.TrimSpace(gitDirRes.Stdout)
		if filepath.IsAbs(gdir) {
			gitDir = gdir
		} else {
			gitDir = filepath.Join(dir, gdir)
		}
	}

	// Step 2: Primary observation via git status --porcelain=v2 --branch
	statusRes, err := r.Run(ctx, dir, "git", "status", "--porcelain=v2", "--branch")
	if err != nil {
		return raw, err
	}

	scanner := bufio.NewScanner(strings.NewReader(statusRes.Stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			// Branch headers
			parts := strings.SplitN(line[2:], " ", 2)
			if len(parts) < 2 {
				continue
			}
			key := parts[0]
			val := strings.TrimSpace(parts[1])

			switch key {
			case "branch.head":
				if val == "(detached)" {
					raw.Detached = true
					raw.CurrentBranch = ""
				} else {
					raw.CurrentBranch = val
				}
			case "branch.upstream":
				if val != "" {
					raw.HasUpstream = true
					raw.Upstream = val
				}
			case "branch.ab":
				// Format: +<ahead> -<behind>
				abFields := strings.Fields(val)
				if len(abFields) >= 2 {
					if strings.HasPrefix(abFields[0], "+") {
						raw.Ahead, _ = strconv.Atoi(abFields[0][1:])
					}
					if strings.HasPrefix(abFields[1], "-") {
						raw.Behind, _ = strconv.Atoi(abFields[1][1:])
					}
				}
			}
		} else if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			// Ordinary or renamed entries: <XY> sub ...
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				xy := fields[1]
				if len(xy) >= 2 {
					if xy[0] != '.' {
						raw.Staged = true
					}
					if xy[1] != '.' {
						raw.Unstaged = true
					}
				}
			}
		} else if strings.HasPrefix(line, "u ") {
			// Unmerged / conflicted entries
			raw.Conflicted = true
		} else if strings.HasPrefix(line, "? ") {
			// Untracked entries
			raw.Untracked = true
		}
	}

	raw.Diverged = raw.Ahead > 0 && raw.Behind > 0

	// Step 3: Check in-progress operations via marker files
	raw.InProgressOp = detectInProgressOp(gitDir)

	// Step 4: Detect default branch
	raw.DefaultBranch = detectDefaultBranch(ctx, r, dir)

	// Step 5: Clean calculation
	raw.Clean = !raw.Staged && !raw.Unstaged && !raw.Untracked && !raw.Conflicted && raw.InProgressOp == ""

	return raw, nil
}

func detectInProgressOp(gitDir string) string {
	if fileExists(filepath.Join(gitDir, "MERGE_HEAD")) {
		return "merge"
	}
	if fileExists(filepath.Join(gitDir, "rebase-merge")) || fileExists(filepath.Join(gitDir, "rebase-apply")) {
		return "rebase"
	}
	if fileExists(filepath.Join(gitDir, "CHERRY_PICK_HEAD")) {
		return "cherry-pick"
	}
	if fileExists(filepath.Join(gitDir, "REVERT_HEAD")) {
		return "revert"
	}
	if fileExists(filepath.Join(gitDir, "BISECT_LOG")) {
		return "bisect"
	}
	return ""
}

func detectDefaultBranch(ctx context.Context, r exec.Runner, dir string) string {
	// Try origin symbolic-ref
	symRes, err := r.Run(ctx, dir, "git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil && symRes.ExitCode == 0 {
		out := strings.TrimSpace(symRes.Stdout)
		if strings.HasPrefix(out, "origin/") {
			return strings.TrimPrefix(out, "origin/")
		}
		if out != "" {
			return out
		}
	}

	// Try checking if main or master exist
	mainRes, _ := r.Run(ctx, dir, "git", "rev-parse", "--verify", "refs/heads/main")
	if mainRes.ExitCode == 0 {
		return "main"
	}
	masterRes, _ := r.Run(ctx, dir, "git", "rev-parse", "--verify", "refs/heads/master")
	if masterRes.ExitCode == 0 {
		return "master"
	}

	return "main"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}