package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gitskill/gw/internal/cache"
	"github.com/gitskill/gw/internal/eventlog"
	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
	"github.com/gitskill/gw/internal/workflow"
)

// Version of the gw engine.
const Version = "0.1.0"

// App manages CLI flag parsing and command dispatch.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Runner exec.Runner
	Cache  *cache.Cache[state.RepoState]
	Logger *eventlog.Logger
}

// New creates an App instance with standard I/O and runner.
func New(stdout, stderr io.Writer, runner exec.Runner) *App {
	if runner == nil {
		runner = exec.RealRunner{Timeout: 30 * time.Second}
	}
	return &App{
		Stdout: stdout,
		Stderr: stderr,
		Runner: runner,
		Cache:  cache.New[state.RepoState](2 * time.Second),
	}
}

// Run parses command line arguments and dispatches to the appropriate workflow.
func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printHelp()
		return 2
	}

	cmd := args[0]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		a.printHelp()
		return 0
	}
	if cmd == "version" || cmd == "--version" || cmd == "-v" {
		fmt.Fprintf(a.Stdout, "gw version %s\n", Version)
		return 0
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	jsonOutput := fs.Bool("json", false, "Output machine-readable JSON envelope")
	debug := fs.Bool("debug", false, "Enable debug trace output to stderr")
	noCache := fs.Bool("no-cache", false, "Bypass read cache")
	yes := fs.Bool("yes", false, "Acknowledge confirmation for guarded actions")
	configPath := fs.String("config", "", "Path to custom configuration file")
	allowRepoConfig := fs.Bool("allow-repo-config", false, "Allow reading repo-local .gw.yml")
	baseBranch := fs.String("base", "", "Target base branch override")
	message := fs.String("message", "", "Commit message for prepare workflow")
	title := fs.String("title", "", "Pull request title")
	body := fs.String("body", "", "Pull request description")
	draft := fs.Bool("draft", false, "Create PR as draft")
	mergeMethod := fs.String("merge-method", "", "Merge strategy: squash, merge, or rebase")
	prNumber := fs.Int("pr", 0, "Specific pull request number")

	// Policy override flags
	commitAllow := fs.Bool("commit-allow", false, "Override commit.allow policy")
	commitRequireConfirmation := fs.Bool("commit-require-confirmation", true, "Override commit.require_confirmation policy")
	pushAllow := fs.Bool("push-allow", false, "Override push.allow policy")
	pushSetUpstream := fs.Bool("push-set-upstream", false, "Override push.allow_set_upstream policy")
	pushRequireConfirmation := fs.Bool("push-require-confirmation", true, "Override push.require_confirmation policy")
	mergeAllow := fs.Bool("merge-allow", false, "Override merge.allow policy")
	syncStrategy := fs.String("sync-strategy", "", "Override sync strategy (e.g. ff-only)")
	syncRequireConfirmation := fs.Bool("sync-require-confirmation", true, "Override sync.require_confirmation policy")

	parsedArgs := normalizeBoolArgs(args[1:])
	if err := fs.Parse(parsedArgs); err != nil {
		env := result.NewEnvelope(
			result.ValidationFailed,
			"INVALID_FLAG",
			cmd,
			err.Error(),
			nil,
			nil,
			nil,
			nil,
		)
		a.outputEnvelope(env, *jsonOutput)
		return env.ExitCode()
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Resolve policy overrides
	var flagBase *string
	var flagCommitAllow *bool
	var flagCommitRequireConfirmation *bool
	var flagPushAllow *bool
	var flagPushSetUpstream *bool
	var flagPushRequireConfirmation *bool
	var flagPRDraft *bool
	var flagMergeAllow *bool
	var flagMergeMethod *string
	var flagSyncStrategy *string
	var flagSyncRequireConfirmation *bool

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "base":
			flagBase = baseBranch
		case "commit-allow":
			flagCommitAllow = commitAllow
		case "commit-require-confirmation":
			flagCommitRequireConfirmation = commitRequireConfirmation
		case "push-allow":
			flagPushAllow = pushAllow
		case "push-set-upstream":
			flagPushSetUpstream = pushSetUpstream
		case "push-require-confirmation":
			flagPushRequireConfirmation = pushRequireConfirmation
		case "draft":
			flagPRDraft = draft
		case "merge-allow":
			flagMergeAllow = mergeAllow
		case "merge-method":
			flagMergeMethod = mergeMethod
		case "sync-strategy":
			flagSyncStrategy = syncStrategy
		case "sync-require-confirmation":
			flagSyncRequireConfirmation = syncRequireConfirmation
		}
	})

	overrides := &policy.PolicyOverrides{
		BaseBranch:                flagBase,
		CommitAllow:               flagCommitAllow,
		CommitRequireConfirmation: flagCommitRequireConfirmation,
		PushAllow:                 flagPushAllow,
		PushSetUpstream:           flagPushSetUpstream,
		PushRequireConfirmation:   flagPushRequireConfirmation,
		PRDraft:                   flagPRDraft,
		MergeAllow:                flagMergeAllow,
		MergeMethod:               flagMergeMethod,
		SyncStrategy:              flagSyncStrategy,
		SyncRequireConfirmation:   flagSyncRequireConfirmation,
		Approved:                  *yes,
	}

	pol, err := policy.Load(policy.LoadOptions{
		ExplicitConfigPath: *configPath,
		AllowRepoConfig:    *allowRepoConfig,
		RepoRoot:           cwd,
		Flags:              overrides,
	})
	if err != nil {
		env := result.NewEnvelope(
			result.ValidationFailed,
			"POLICY_CONFIG_ERROR",
			cmd,
			err.Error(),
			nil,
			nil,
			nil,
			nil,
		)
		a.outputEnvelope(env, *jsonOutput)
		return env.ExitCode()
	}

	logger := a.Logger
	if logger == nil {
		gitDir := filepath.Join(cwd, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			logger = eventlog.NewLogger(filepath.Join(gitDir, "gw.events.jsonl"), eventlog.DefaultMaxSize)
		}
	}

	params := workflow.Params{
		Dir:           cwd,
		Runner:        a.Runner,
		Policy:        pol,
		EngineVersion: Version,
		Cache:         a.Cache,
		Logger:        logger,
		NoCache:       *noCache,
		Debug:         *debug,
		BaseBranch:    *baseBranch,
		CommitMessage: *message,
		PRNumber:      *prNumber,
		PRTitle:       *title,
		PRBody:        *body,
		PRDraft:       *draft,
		MergeMethod:   pol.Merge.Method,
	}

	var env result.Envelope

	switch cmd {
	case "inspect":
		env = workflow.Inspect(ctx, params)
	case "prepare":
		env = workflow.Prepare(ctx, params)
	case "push":
		env = workflow.Push(ctx, params)
	case "sync":
		env = workflow.Sync(ctx, params)
	case "pr-ready":
		env = workflow.PRReady(ctx, params)
	case "pr-create":
		env = workflow.PRCreate(ctx, params)
	case "pr-status":
		env = workflow.PRStatus(ctx, params)
	case "pr-merge":
		env = workflow.PRMerge(ctx, params)
	case "doctor":
		env = workflow.Doctor(ctx, params)
	default:
		env = result.NewEnvelope(
			result.ValidationFailed,
			"USAGE_ERROR",
			cmd,
			fmt.Sprintf("unknown command %q", cmd),
			nil,
			nil,
			nil,
			nil,
		)
	}

	a.outputEnvelope(env, *jsonOutput)
	return env.ExitCode()
}

func (a *App) outputEnvelope(env result.Envelope, asJSON bool) {
	if asJSON {
		data, _ := env.ToJSON()
		fmt.Fprintf(a.Stdout, "%s\n", string(data))
		return
	}

	// Human-readable format
	statusPrefix := "[" + string(env.Status) + "]"
	fmt.Fprintf(a.Stdout, "%-16s %s: %s\n", statusPrefix, env.Code, env.Reason)
	if env.NextAction != nil {
		fmt.Fprintf(a.Stdout, "  Next action:   %s\n", *env.NextAction)
	}
	if env.RetryAfterHint != nil {
		fmt.Fprintf(a.Stdout, "  Retry after:   %ds\n", *env.RetryAfterHint)
	}
}

func (a *App) printHelp() {
	helpText := `gw - GitHub-Aware Deterministic Git Workflow Engine

Usage:
  gw <command> [flags]

Workflows:
  inspect    Observe and normalize current Git and GitHub state
  prepare    Stage and commit changes (if policy allows)
  push       Push current branch to remote upstream
  sync       Fast-forward sync current branch with upstream
  pr-ready   Evaluate if current branch is ready for pull request
  pr-create  Create a pull request with preconditions checked
  pr-status  Inspect pull request and CI status
  pr-merge   Guarded pull request merge with 2-factor verification
  doctor     Diagnostic check for git, gh, and auth status
  version    Show engine version

Global Flags:
  --json               Output machine-readable JSON envelope
  --debug              Enable stderr debug tracing
  --no-cache           Bypass in-process observation cache
  --yes                User confirmation for guarded mutations
  --config <path>      Path to custom configuration file
  --allow-repo-config  Opt-in to load .gw.yml from repository
  --base <branch>      Target base branch override
  --commit-allow       Override commit.allow policy
  --commit-require-confirmation  Override commit.require_confirmation policy
  --push-allow         Override push.allow policy
  --push-set-upstream  Override push.allow_set_upstream policy
  --push-require-confirmation    Override push.require_confirmation policy
  --merge-allow        Override merge.allow policy
  --merge-method <m>   Override merge strategy (squash, merge, rebase)
  --sync-strategy <s>  Override sync strategy (ff-only)
  --sync-require-confirmation    Override sync.require_confirmation policy
`
	fmt.Fprintf(a.Stdout, "%s\n", strings.TrimSpace(helpText))
}

func normalizeBoolArgs(args []string) []string {
	boolFlags := map[string]bool{
		"json":                        true,
		"debug":                       true,
		"no-cache":                    true,
		"yes":                         true,
		"allow-repo-config":           true,
		"draft":                       true,
		"commit-allow":                true,
		"commit-require-confirmation": true,
		"push-allow":                  true,
		"push-set-upstream":           true,
		"push-require-confirmation":   true,
		"merge-allow":                 true,
		"sync-require-confirmation":   true,
	}

	var res []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		trimmed := strings.TrimLeft(arg, "-")
		if boolFlags[trimmed] && i+1 < len(args) {
			next := strings.ToLower(args[i+1])
			if next == "true" || next == "false" {
				res = append(res, arg+"="+next)
				i++
				continue
			}
		}
		res = append(res, arg)
	}
	return res
}