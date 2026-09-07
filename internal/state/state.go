package state

import (
	"encoding/json"
	"time"

	"github.com/gitskill/gw/internal/ghstate"
	"github.com/gitskill/gw/internal/gitstate"
)

// CurrentSchemaVersion defines the schema version for RepoState.
const CurrentSchemaVersion = 1

// RepositoryState represents normalized Git repository identity and layout.
type RepositoryState struct {
	IsGitRepo     bool   `json:"is_git_repo"`
	Root          string `json:"root"`
	DefaultBranch string `json:"default_branch"`
	CurrentBranch string `json:"current_branch"`
	Detached      bool   `json:"detached"`
}

// WorkingTreeState represents normalized working tree modification status.
type WorkingTreeState struct {
	Clean               bool    `json:"clean"`
	Staged              bool    `json:"staged"`
	Unstaged            bool    `json:"unstaged"`
	Untracked           bool    `json:"untracked"`
	Conflicted          bool    `json:"conflicted"`
	InProgressOperation *string `json:"in_progress_operation"`
}

// BranchState represents normalized branch tracking, divergence, and ahead/behind metrics.
type BranchState struct {
	IsDefault   bool   `json:"is_default"`
	HasUpstream bool   `json:"has_upstream"`
	Upstream    string `json:"upstream"`
	Ahead       int    `json:"ahead"`
	Behind      int    `json:"behind"`
	Diverged    bool   `json:"diverged"`
}

// GitHubState represents normalized GitHub CLI availability, auth, and PR status.
type GitHubState struct {
	Available     bool                 `json:"available"`
	Authenticated bool                 `json:"authenticated"`
	Repository    string               `json:"repository"`
	PullRequest   *ghstate.PullRequest `json:"pull_request"`
}

// RepoState is the unified, normalized snapshot of Git and GitHub status.
type RepoState struct {
	SchemaVersion int              `json:"schema_version"`
	ObservedAt    time.Time        `json:"observed_at"`
	EngineVersion string           `json:"engine_version"`
	Repository    RepositoryState  `json:"repository"`
	WorkingTree   WorkingTreeState `json:"working_tree"`
	Branch        BranchState      `json:"branch"`
	GitHub        GitHubState      `json:"github"`
}

// Normalize combines raw Git and GitHub observation data into a pure RepoState struct.
// Enforces the "missing != false" rule: PullRequest is nil unless gh is authenticated and PR exists.
func Normalize(g gitstate.Raw, gh ghstate.Raw, engineVersion string, now time.Time) RepoState {
	var inProgress *string
	if g.InProgressOp != "" {
		op := g.InProgressOp
		inProgress = &op
	}

	var pr *ghstate.PullRequest
	if gh.Authenticated && gh.PullRequest != nil {
		pr = gh.PullRequest
	}

	isDefault := false
	if g.IsRepo && !g.Detached && g.CurrentBranch != "" && g.CurrentBranch == g.DefaultBranch {
		isDefault = true
	}

	return RepoState{
		SchemaVersion: CurrentSchemaVersion,
		ObservedAt:    now.UTC(),
		EngineVersion: engineVersion,
		Repository: RepositoryState{
			IsGitRepo:     g.IsRepo,
			Root:          g.Root,
			DefaultBranch: g.DefaultBranch,
			CurrentBranch: g.CurrentBranch,
			Detached:      g.Detached,
		},
		WorkingTree: WorkingTreeState{
			Clean:               g.Clean,
			Staged:              g.Staged,
			Unstaged:            g.Unstaged,
			Untracked:           g.Untracked,
			Conflicted:          g.Conflicted,
			InProgressOperation: inProgress,
		},
		Branch: BranchState{
			IsDefault:   isDefault,
			HasUpstream: g.HasUpstream,
			Upstream:    g.Upstream,
			Ahead:       g.Ahead,
			Behind:      g.Behind,
			Diverged:    g.Diverged,
		},
		GitHub: GitHubState{
			Available:     gh.Available,
			Authenticated: gh.Authenticated,
			Repository:    gh.Repository,
			PullRequest:   pr,
		},
	}
}

// ToJSON serializes RepoState into indented JSON.
func (s RepoState) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}