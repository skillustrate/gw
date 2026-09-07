package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gitskill/gw/internal/ghstate"
	"github.com/gitskill/gw/internal/gitstate"
)

func boolPtr(b bool) *bool { return &b }

// pullRequestsEqual compares two PullRequest values by content, dereferencing
// the Mergeable pointer instead of comparing it by address (struct != would
// otherwise compare Mergeable's pointer identity, not its pointed-to value).
func pullRequestsEqual(a, b *ghstate.PullRequest) bool {
	if a.Mergeable == nil || b.Mergeable == nil {
		if a.Mergeable != b.Mergeable {
			return false
		}
	} else if *a.Mergeable != *b.Mergeable {
		return false
	}

	return a.Number == b.Number &&
		a.URL == b.URL &&
		a.State == b.State &&
		a.Draft == b.Draft &&
		a.ChecksStatus == b.ChecksStatus &&
		a.ReviewDecision == b.ReviewDecision
}

func TestNormalize_GoldenComparison(t *testing.T) {
	fixedTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	engineVersion := "0.1.0"

	cases := []struct {
		name     string
		golden   string
		gitRaw   gitstate.Raw
		ghRaw    ghstate.Raw
	}{
		{
			name:   "Clean with open PR",
			golden: "clean_with_pr.json",
			gitRaw: gitstate.Raw{
				IsRepo:        true,
				Root:          "/home/user/project",
				DefaultBranch: "main",
				CurrentBranch: "feature/login",
				Clean:         true,
				HasUpstream:   true,
				Upstream:      "origin/feature/login",
			},
			ghRaw: ghstate.Raw{
				Available:     true,
				Authenticated: true,
				Repository:    "org/project",
				PullRequest: &ghstate.PullRequest{
					Number:         42,
					URL:            "https://github.com/org/project/pull/42",
					State:          "OPEN",
					Draft:          false,
					Mergeable:      boolPtr(true),
					ChecksStatus:   "passing",
					ReviewDecision: "APPROVED",
				},
			},
		},
		{
			name:   "Unauthenticated - Missing != False Rule",
			golden: "unauthenticated_no_pr.json",
			gitRaw: gitstate.Raw{
				IsRepo:        true,
				Root:          "/home/user/project",
				DefaultBranch: "main",
				CurrentBranch: "feature/login",
				Staged:        true,
				Clean:         false,
			},
			ghRaw: ghstate.Raw{
				Available:     true,
				Authenticated: false,
				Repository:    "",
				PullRequest:   nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := Normalize(tc.gitRaw, tc.ghRaw, engineVersion, fixedTime)

			path := filepath.Join("testdata", "golden", tc.golden)
			expectedBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", path, err)
			}

			var expectedState RepoState
			if err := json.Unmarshal(expectedBytes, &expectedState); err != nil {
				t.Fatalf("failed to unmarshal golden file %s: %v", tc.golden, err)
			}

			if normalized.Repository != expectedState.Repository {
				t.Errorf("Repository mismatch:\ngot  %+v\nwant %+v", normalized.Repository, expectedState.Repository)
			}
			if normalized.WorkingTree.Clean != expectedState.WorkingTree.Clean ||
				normalized.WorkingTree.Staged != expectedState.WorkingTree.Staged ||
				normalized.WorkingTree.Unstaged != expectedState.WorkingTree.Unstaged ||
				normalized.WorkingTree.Untracked != expectedState.WorkingTree.Untracked ||
				normalized.WorkingTree.Conflicted != expectedState.WorkingTree.Conflicted {
				t.Errorf("WorkingTree mismatch:\ngot  %+v\nwant %+v", normalized.WorkingTree, expectedState.WorkingTree)
			}
			if normalized.Branch != expectedState.Branch {
				t.Errorf("Branch mismatch:\ngot  %+v\nwant %+v", normalized.Branch, expectedState.Branch)
			}
			if normalized.GitHub.Available != expectedState.GitHub.Available ||
				normalized.GitHub.Authenticated != expectedState.GitHub.Authenticated ||
				normalized.GitHub.Repository != expectedState.GitHub.Repository {
				t.Errorf("GitHub metadata mismatch:\ngot  %+v\nwant %+v", normalized.GitHub, expectedState.GitHub)
			}

			if expectedState.GitHub.PullRequest == nil && normalized.GitHub.PullRequest != nil {
				t.Errorf("expected PullRequest to be nil, got %+v", normalized.GitHub.PullRequest)
			}
			if expectedState.GitHub.PullRequest != nil {
				if normalized.GitHub.PullRequest == nil {
					t.Fatalf("expected non-nil PullRequest")
				}
				if !pullRequestsEqual(normalized.GitHub.PullRequest, expectedState.GitHub.PullRequest) {
					t.Errorf("PullRequest mismatch:\ngot  %+v\nwant %+v", *normalized.GitHub.PullRequest, *expectedState.GitHub.PullRequest)
				}
			}
		})
	}
}