package ghstate

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gitskill/gw/internal/exec"
)

// PullRequest encapsulates normalized GitHub pull request state.
type PullRequest struct {
	Number         int    `json:"number"`
	URL            string `json:"url"`
	State          string `json:"state"` // OPEN, CLOSED, MERGED
	Draft          bool   `json:"draft"`
	Mergeable      *bool  `json:"mergeable"` // nil = unknown
	ChecksStatus   string `json:"checks_status"` // pending, passing, failing, none
	ReviewDecision string `json:"review_decision"` // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, ""
}

// Raw contains observed GitHub CLI availability, authentication, and PR state.
type Raw struct {
	Available     bool
	Authenticated bool
	Repository    string
	PullRequest   *PullRequest
}

type ghPRViewOutput struct {
	Number            int    `json:"number"`
	URL               string `json:"url"`
	State             string `json:"state"`
	IsDraft           bool   `json:"isDraft"`
	Mergeable         string `json:"mergeable"`
	ReviewDecision    string `json:"reviewDecision"`
	StatusCheckRollup []struct {
		State      string `json:"state"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	} `json:"statusCheckRollup"`
}

type ghRepoViewOutput struct {
	NameWithOwner string `json:"nameWithOwner"`
}

// Observe inspects GitHub state for the specified branch via gh CLI.
func Observe(ctx context.Context, r exec.Runner, dir, branch string) (Raw, error) {
	// Step 1: Check gh binary availability and auth status
	authRes, err := r.Run(ctx, dir, "gh", "auth", "status")
	if err != nil && authRes.ExitCode == -1 {
		// gh binary not found or failed to execute
		return Raw{Available: false, Authenticated: false}, nil
	}

	if authRes.ExitCode != 0 {
		// gh is installed but not authenticated
		return Raw{Available: true, Authenticated: false}, nil
	}

	raw := Raw{
		Available:     true,
		Authenticated: true,
	}

	// Step 2: Fetch repo nameWithOwner
	repoRes, err := r.Run(ctx, dir, "gh", "repo", "view", "--json", "nameWithOwner")
	if err == nil && repoRes.ExitCode == 0 {
		var repoOut ghRepoViewOutput
		if jsonErr := json.Unmarshal([]byte(repoRes.Stdout), &repoOut); jsonErr == nil {
			raw.Repository = repoOut.NameWithOwner
		}
	}

	if branch == "" {
		return raw, nil
	}

	// Step 3: Single batched read for branch PR + checks + reviews
	prRes, err := r.Run(ctx, dir, "gh", "pr", "view", branch, "--json", "number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision")
	if err != nil || prRes.ExitCode != 0 {
		// No PR exists for this branch or command failed
		raw.PullRequest = nil
		return raw, nil
	}

	var prOut ghPRViewOutput
	if jsonErr := json.Unmarshal([]byte(prRes.Stdout), &prOut); jsonErr != nil || prOut.Number == 0 {
		raw.PullRequest = nil
		return raw, nil
	}

	pr := &PullRequest{
		Number:         prOut.Number,
		URL:            prOut.URL,
		State:          strings.ToUpper(prOut.State),
		Draft:          prOut.IsDraft,
		ReviewDecision: strings.ToUpper(prOut.ReviewDecision),
		ChecksStatus:   evaluateChecksStatus(prOut.StatusCheckRollup),
	}

	switch strings.ToUpper(prOut.Mergeable) {
	case "MERGEABLE":
		val := true
		pr.Mergeable = &val
	case "CONFLICTING":
		val := false
		pr.Mergeable = &val
	default:
		pr.Mergeable = nil
	}

	raw.PullRequest = pr
	return raw, nil
}

func evaluateChecksStatus(checks []struct {
	State      string `json:"state"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}) string {
	if len(checks) == 0 {
		return "none"
	}

	hasPending := false
	for _, c := range checks {
		st := strings.ToUpper(c.State)
		status := strings.ToUpper(c.Status)
		conc := strings.ToUpper(c.Conclusion)

		if st == "FAILURE" || st == "ERROR" || conc == "FAILURE" || conc == "TIMED_OUT" || conc == "CANCELLED" {
			return "failing"
		}
		if st == "PENDING" || status == "IN_PROGRESS" || status == "QUEUED" || conc == "ACTION_REQUIRED" {
			hasPending = true
		}
	}

	if hasPending {
		return "pending"
	}

	return "passing"
}