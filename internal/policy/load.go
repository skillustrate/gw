package policy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LoadOptions configures discovery and precedence rules for policy resolution.
type LoadOptions struct {
	ExplicitConfigPath string
	AllowRepoConfig    bool
	RepoRoot           string
	Env                map[string]string
	Flags              *PolicyOverrides
}

// PolicyOverrides specifies explicit CLI flag overrides with pointer semantics.
type PolicyOverrides struct {
	BaseBranch                 *string
	CommitAllow                *bool
	CommitRequireConfirmation  *bool
	PushAllow                  *bool
	PushSetUpstream            *bool
	PushRequireConfirmation    *bool
	PRCreate                   *bool
	PRDraft                    *bool
	MergeAllow                 *bool
	MergeRequireChecks         *bool
	MergeRequireApprove        *bool
	MergeMethod                *string
	SyncStrategy               *string
	SyncRequireConfirmation    *bool
	LockStaleAfter             *time.Duration
	Approved                   bool
}

// Load discovers, parses, validates, and resolves the final policy across precedence layers.
// Precedence: Defaults < Config File < Environment Variables < CLI Flags.
func Load(opts LoadOptions) (Policy, error) {
	p := Default()

	// Step 1: Discover configuration file path
	configPath := discoverConfigFile(opts)
	if configPath != "" {
		if err := loadYAMLFile(configPath, &p); err != nil {
			return p, fmt.Errorf("failed to load config %q: %w", configPath, err)
		}
	}

	// Step 2: Environment variable overrides
	applyEnvOverrides(opts.Env, &p)

	// Step 3: CLI Flag overrides (highest precedence)
	if opts.Flags != nil {
		applyFlagOverrides(opts.Flags, &p)
	}

	// Step 4: Validate resolved policy
	if err := validatePolicy(p); err != nil {
		return p, err
	}

	return p, nil
}

func discoverConfigFile(opts LoadOptions) string {
	if opts.ExplicitConfigPath != "" {
		return opts.ExplicitConfigPath
	}

	// Repo-local config ONLY if explicitly allowed
	if opts.AllowRepoConfig && opts.RepoRoot != "" {
		for _, name := range []string{".gw.yml", ".gw.yaml"} {
			candidate := filepath.Join(opts.RepoRoot, name)
			if fileExists(candidate) {
				return candidate
			}
		}
	}

	// User global config: ~/.config/gw/config.yaml or %USERPROFILE%/.config/gw/config.yaml
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		candidate := filepath.Join(home, ".config", "gw", "config.yaml")
		if fileExists(candidate) {
			return candidate
		}
		candidateYml := filepath.Join(home, ".config", "gw", "config.yml")
		if fileExists(candidateYml) {
			return candidateYml
		}
	}

	return ""
}

func loadYAMLFile(path string, p *Policy) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		trimmed := strings.TrimSpace(line)

		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check indent level
		indent := len(line) - len(strings.TrimLeft(line, " "))

		if indent == 0 {
			// Top-level key or section
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			val := ""
			if len(parts) > 1 {
				val = strings.TrimSpace(parts[1])
			}

			if val == "" {
				// Section header
				switch key {
				case "commit", "push", "pull_request", "merge", "sync", "lock":
					currentSection = key
				default:
					return fmt.Errorf("unknown config section: %q", key)
				}
			} else {
				currentSection = ""
				switch key {
				case "version":
					v, err := strconv.Atoi(val)
					if err != nil {
						return fmt.Errorf("invalid version value: %q", val)
					}
					p.Version = v
				case "base_branch":
					p.BaseBranch = val
				default:
					return fmt.Errorf("unknown top-level key: %q", key)
				}
			}
		} else {
			// Sub-key in current section
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) < 2 {
				return fmt.Errorf("malformed yaml line: %q", trimmed)
			}
			subKey := strings.TrimSpace(parts[0])
			subVal := strings.TrimSpace(parts[1])

			switch currentSection {
			case "commit":
				switch subKey {
				case "allow":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid commit.allow boolean: %q", subVal)
					}
					p.Commit.Allow = b
				case "require_confirmation":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid commit.require_confirmation: %q", subVal)
					}
					p.Commit.RequireConfirmation = b
				default:
					return fmt.Errorf("unknown key under commit: %q", subKey)
				}
			case "push":
				switch subKey {
				case "allow":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid push.allow: %q", subVal)
					}
					p.Push.Allow = b
				case "allow_set_upstream":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid push.allow_set_upstream: %q", subVal)
					}
					p.Push.AllowSetUpstream = b
				case "require_confirmation":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid push.require_confirmation: %q", subVal)
					}
					p.Push.RequireConfirmation = b
				default:
					return fmt.Errorf("unknown key under push: %q", subKey)
				}
			case "pull_request":
				switch subKey {
				case "create":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid pull_request.create: %q", subVal)
					}
					p.PullRequest.Create = b
				case "draft":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid pull_request.draft: %q", subVal)
					}
					p.PullRequest.Draft = b
				default:
					return fmt.Errorf("unknown key under pull_request: %q", subKey)
				}
			case "merge":
				switch subKey {
				case "allow":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid merge.allow: %q", subVal)
					}
					p.Merge.Allow = b
				case "require_checks":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid merge.require_checks: %q", subVal)
					}
					p.Merge.RequireChecks = b
				case "require_approval":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid merge.require_approval: %q", subVal)
					}
					p.Merge.RequireApproval = b
				case "method":
					p.Merge.Method = subVal
				default:
					return fmt.Errorf("unknown key under merge: %q", subKey)
				}
			case "sync":
				switch subKey {
				case "strategy":
					p.Sync.Strategy = subVal
				case "require_confirmation":
					b, err := strconv.ParseBool(subVal)
					if err != nil {
						return fmt.Errorf("invalid sync.require_confirmation: %q", subVal)
					}
					p.Sync.RequireConfirmation = b
				default:
					return fmt.Errorf("unknown key under sync: %q", subKey)
				}
			case "lock":
				if subKey == "stale_after" {
					dur, err := time.ParseDuration(subVal)
					if err != nil {
						return fmt.Errorf("invalid lock.stale_after duration: %q", subVal)
					}
					p.Lock.StaleAfter = dur
				} else {
					return fmt.Errorf("unknown key under lock: %q", subKey)
				}
			default:
				return fmt.Errorf("orphaned key without section: %q", subKey)
			}
		}
	}

	return scanner.Err()
}

func applyEnvOverrides(env map[string]string, p *Policy) {
	if env == nil {
		return
	}
	if v, ok := env["GW_BASE_BRANCH"]; ok && v != "" {
		p.BaseBranch = v
	}
	if v, ok := env["GW_COMMIT_ALLOW"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Commit.Allow = b
		}
	}
	if v, ok := env["GW_COMMIT_REQUIRE_CONFIRMATION"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Commit.RequireConfirmation = b
		}
	}
	if v, ok := env["GW_PUSH_ALLOW"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Push.Allow = b
		}
	}
	if v, ok := env["GW_PUSH_SET_UPSTREAM"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Push.AllowSetUpstream = b
		}
	}
	if v, ok := env["GW_PUSH_REQUIRE_CONFIRMATION"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Push.RequireConfirmation = b
		}
	}
	if v, ok := env["GW_PR_CREATE"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.PullRequest.Create = b
		}
	}
	if v, ok := env["GW_PR_DRAFT"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.PullRequest.Draft = b
		}
	}
	if v, ok := env["GW_MERGE_ALLOW"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Merge.Allow = b
		}
	}
	if v, ok := env["GW_MERGE_METHOD"]; ok && v != "" {
		p.Merge.Method = v
	}
	if v, ok := env["GW_SYNC_STRATEGY"]; ok && v != "" {
		p.Sync.Strategy = v
	}
	if v, ok := env["GW_SYNC_REQUIRE_CONFIRMATION"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			p.Sync.RequireConfirmation = b
		}
	}
}

func applyFlagOverrides(f *PolicyOverrides, p *Policy) {
	if f.BaseBranch != nil {
		p.BaseBranch = *f.BaseBranch
	}
	if f.CommitAllow != nil {
		p.Commit.Allow = *f.CommitAllow
	}
	if f.CommitRequireConfirmation != nil {
		p.Commit.RequireConfirmation = *f.CommitRequireConfirmation
	}
	if f.PushAllow != nil {
		p.Push.Allow = *f.PushAllow
	}
	if f.PushSetUpstream != nil {
		p.Push.AllowSetUpstream = *f.PushSetUpstream
	}
	if f.PushRequireConfirmation != nil {
		p.Push.RequireConfirmation = *f.PushRequireConfirmation
	}
	if f.PRCreate != nil {
		p.PullRequest.Create = *f.PRCreate
	}
	if f.PRDraft != nil {
		p.PullRequest.Draft = *f.PRDraft
	}
	if f.MergeAllow != nil {
		p.Merge.Allow = *f.MergeAllow
	}
	if f.MergeRequireChecks != nil {
		p.Merge.RequireChecks = *f.MergeRequireChecks
	}
	if f.MergeRequireApprove != nil {
		p.Merge.RequireApproval = *f.MergeRequireApprove
	}
	if f.MergeMethod != nil {
		p.Merge.Method = *f.MergeMethod
	}
	if f.SyncStrategy != nil {
		p.Sync.Strategy = *f.SyncStrategy
	}
	if f.SyncRequireConfirmation != nil {
		p.Sync.RequireConfirmation = *f.SyncRequireConfirmation
	}
	if f.LockStaleAfter != nil {
		p.Lock.StaleAfter = *f.LockStaleAfter
	}
	p.Approved = f.Approved
}

func validatePolicy(p Policy) error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported policy schema version: %d (expected 1)", p.Version)
	}
	if p.BaseBranch == "" {
		return fmt.Errorf("base_branch cannot be empty")
	}
	if p.Merge.Method != "" && p.Merge.Method != "squash" && p.Merge.Method != "merge" && p.Merge.Method != "rebase" {
		return fmt.Errorf("unsupported merge.method: %q (expected 'squash', 'merge', or 'rebase')", p.Merge.Method)
	}
	if p.Sync.Strategy != "ff-only" {
		return fmt.Errorf("unsupported sync.strategy: %q (only 'ff-only' supported in MVP)", p.Sync.Strategy)
	}
	if p.Lock.StaleAfter <= 0 {
		return fmt.Errorf("lock.stale_after must be positive duration")
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}