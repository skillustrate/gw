package ghstate

import (
	"context"
	"testing"

	"github.com/gitskill/gw/internal/exec"
)

func TestObserve_GhUnavailable(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{})
	raw, err := Observe(context.Background(), fake, "/repo", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.Available || raw.Authenticated {
		t.Errorf("expected Available=false, Authenticated=false, got %+v", raw)
	}
	if raw.PullRequest != nil {
		t.Errorf("expected PullRequest=nil when unavailable")
	}
}

func TestObserve_GhUnauthenticated(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"gh auth status": {
			ExitCode: 1,
			Stderr:   "You are not logged into any GitHub hosts.",
		},
	})

	raw, err := Observe(context.Background(), fake, "/repo", "feature-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !raw.Available || raw.Authenticated {
		t.Errorf("expected Available=true, Authenticated=false, got %+v", raw)
	}
	if raw.PullRequest != nil {
		t.Errorf("expected PullRequest=nil when unauthenticated (Missing != False rule)")
	}
}

func TestObserve_AuthenticatedWithOpenPR(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"gh auth status": {
			ExitCode: 0,
			Stdout:   "Logged in to github.com as user\n",
		},
		"gh repo view --json nameWithOwner": {
			ExitCode: 0,
			Stdout:   `{"nameWithOwner":"acme/engine"}` + "\n",
		},
		"gh pr view feature-login --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 0,
			Stdout: `{
  "number": 88,
  "url": "https://github.com/acme/engine/pull/88",
  "state": "OPEN",
  "isDraft": false,
  "mergeable": "MERGEABLE",
  "reviewDecision": "APPROVED",
  "statusCheckRollup": [
    {"state": "SUCCESS", "status": "COMPLETED", "conclusion": "SUCCESS"}
  ]
}` + "\n",
		},
	})

	raw, err := Observe(context.Background(), fake, "/repo", "feature-login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !raw.Available || !raw.Authenticated {
		t.Errorf("expected Available=true, Authenticated=true")
	}
	if raw.Repository != "acme/engine" {
		t.Errorf("Repository = %q, want 'acme/engine'", raw.Repository)
	}
	if raw.PullRequest == nil {
		t.Fatalf("expected non-nil PullRequest")
	}
	if raw.PullRequest.Number != 88 {
		t.Errorf("PR Number = %d, want 88", raw.PullRequest.Number)
	}
	if raw.PullRequest.Mergeable == nil || !*raw.PullRequest.Mergeable {
		t.Errorf("expected Mergeable = true")
	}
	if raw.PullRequest.ChecksStatus != "passing" {
		t.Errorf("ChecksStatus = %q, want 'passing'", raw.PullRequest.ChecksStatus)
	}
	if raw.PullRequest.ReviewDecision != "APPROVED" {
		t.Errorf("ReviewDecision = %q, want 'APPROVED'", raw.PullRequest.ReviewDecision)
	}
}

func TestObserve_AuthenticatedNoPR(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"gh auth status": {
			ExitCode: 0,
			Stdout:   "Logged in\n",
		},
		"gh repo view --json nameWithOwner": {
			ExitCode: 0,
			Stdout:   `{"nameWithOwner":"acme/engine"}` + "\n",
		},
		"gh pr view feature-new --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 1,
			Stderr:   "no pull requests found for branch \"feature-new\"\n",
		},
	})

	raw, err := Observe(context.Background(), fake, "/repo", "feature-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if raw.PullRequest != nil {
		t.Errorf("expected PullRequest = nil for branch without PR")
	}
}