package policy

import (
	"time"
)

// CommitPolicy configures whether automated staging and commit operations are permitted.
type CommitPolicy struct {
	Allow               bool `yaml:"allow" json:"allow"`
	RequireConfirmation bool `yaml:"require_confirmation" json:"require_confirmation"`
}

// PushPolicy configures remote push and upstream tracking rules.
type PushPolicy struct {
	Allow               bool `yaml:"allow" json:"allow"`
	AllowSetUpstream    bool `yaml:"allow_set_upstream" json:"allow_set_upstream"`
	RequireConfirmation bool `yaml:"require_confirmation" json:"require_confirmation"`
}

// PullRequestPolicy configures automated pull request creation behaviors.
type PullRequestPolicy struct {
	Create bool `yaml:"create" json:"create"`
	Draft  bool `yaml:"draft" json:"draft"`
}

// MergePolicy configures guarded PR merge criteria.
type MergePolicy struct {
	Allow           bool   `yaml:"allow" json:"allow"`
	RequireChecks   bool   `yaml:"require_checks" json:"require_checks"`
	RequireApproval bool   `yaml:"require_approval" json:"require_approval"`
	Method          string `yaml:"method" json:"method"`
}

// SyncPolicy configures branch synchronization strategy.
type SyncPolicy struct {
	Strategy            string `yaml:"strategy" json:"strategy"`
	RequireConfirmation bool   `yaml:"require_confirmation" json:"require_confirmation"`
}

// LockPolicy configures mutex lock expiration.
type LockPolicy struct {
	StaleAfter time.Duration `yaml:"stale_after" json:"stale_after"`
}

// Policy defines the resolved operational guardrails for all gw workflows.
type Policy struct {
	Version     int               `yaml:"version" json:"version"`
	BaseBranch  string            `yaml:"base_branch" json:"base_branch"`
	Commit      CommitPolicy      `yaml:"commit" json:"commit"`
	Push        PushPolicy        `yaml:"push" json:"push"`
	PullRequest PullRequestPolicy `yaml:"pull_request" json:"pull_request"`
	Merge       MergePolicy       `yaml:"merge" json:"merge"`
	Sync        SyncPolicy        `yaml:"sync" json:"sync"`
	Lock        LockPolicy        `yaml:"lock" json:"lock"`
	Approved    bool              `yaml:"-" json:"approved"` // Set from --yes CLI flag
}

// Default returns the hardcoded safe default policy per §18.
func Default() Policy {
	return Policy{
		Version:    1,
		BaseBranch: "main",
		Commit: CommitPolicy{
			Allow:               false,
			RequireConfirmation: true,
		},
		Push: PushPolicy{
			Allow:               true,
			AllowSetUpstream:    true,
			RequireConfirmation: true,
		},
		PullRequest: PullRequestPolicy{
			Create: true,
			Draft:  false,
		},
		Merge: MergePolicy{
			Allow:           false,
			RequireChecks:   true,
			RequireApproval: true,
			Method:          "squash",
		},
		Sync: SyncPolicy{
			Strategy:            "ff-only",
			RequireConfirmation: true,
		},
		Lock: LockPolicy{
			StaleAfter: 10 * time.Minute,
		},
		Approved: false,
	}
}