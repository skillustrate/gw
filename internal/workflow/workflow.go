package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gitskill/gw/internal/cache"
	"github.com/gitskill/gw/internal/eventlog"
	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/ghstate"
	"github.com/gitskill/gw/internal/gitstate"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// Params contains inputs and options passed from the CLI layer into workflow executions.
type Params struct {
	Dir           string
	Runner        exec.Runner
	Policy        policy.Policy
	EngineVersion string
	Cache         *cache.Cache[state.RepoState]
	Logger        *eventlog.Logger
	NoCache       bool
	Debug         bool

	// Command-specific parameters
	BaseBranch    string
	CommitMessage string
	PRNumber      int
	PRTitle       string
	PRBody        string
	PRDraft       bool
	MergeMethod   string // "merge", "squash", "rebase"
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func policyFingerprint(pol policy.Policy) string {
	return fmt.Sprintf("base:%s,c:%t,p:%t,m:%t,meth:%s,s:%s",
		pol.BaseBranch, pol.Commit.Allow, pol.Push.Allow, pol.Merge.Allow, pol.Merge.Method, pol.Sync.Strategy)
}

func makeCacheKey(dir string, branchOverride string, p Params) string {
	cleanDir := filepath.Clean(dir)
	branch := branchOverride
	if branch == "" {
		branch = "HEAD"
	}
	base := p.BaseBranch
	if base == "" {
		base = p.Policy.BaseBranch
	}
	return fmt.Sprintf("%s|branch:%s|base:%s|pol:%s", cleanDir, branch, base, policyFingerprint(p.Policy))
}

func invalidateCache(p Params) {
	if p.Cache != nil {
		p.Cache.InvalidatePrefix(filepath.Clean(p.Dir) + "|")
	}
}

// observeState inspects git and gh state, utilizing cache where applicable.
func observeState(ctx context.Context, p Params, branchOverride string) (state.RepoState, error) {
	cacheKey := makeCacheKey(p.Dir, branchOverride, p)
	if !p.NoCache && p.Cache != nil {
		if cached, ok := p.Cache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	gitRaw, err := gitstate.Observe(ctx, p.Runner, p.Dir)
	if err != nil {
		return state.RepoState{}, fmt.Errorf("failed to observe git state: %w", err)
	}

	targetBranch := gitRaw.CurrentBranch
	if branchOverride != "" {
		targetBranch = branchOverride
	}

	ghRaw, err := ghstate.Observe(ctx, p.Runner, p.Dir, targetBranch)
	if err != nil {
		return state.RepoState{}, fmt.Errorf("failed to observe github state: %w", err)
	}

	now := time.Now().UTC()
	st := state.Normalize(gitRaw, ghRaw, p.EngineVersion, now)

	if p.Cache != nil {
		p.Cache.Set(cacheKey, st, now)
	}

	return st, nil
}

func logAndBuildEnvelope(p Params, command string, res result.Envelope, reclaimed bool) result.Envelope {
	if p.Logger != nil {
		_ = p.Logger.Log(eventlog.Event{
			Timestamp: time.Now().UTC(),
			Command:   command,
			Status:    string(res.Status),
			Code:      res.Code,
			Reclaimed: reclaimed,
		})
	}
	return res
}