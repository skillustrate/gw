package workflow

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gitskill/gw/internal/result"
)

// DoctorReport contains diagnostic environment status.
type DoctorReport struct {
	GitAvailable bool   `json:"git_available"`
	GitVersion   string `json:"git_version,omitempty"`
	GHAvailable  bool   `json:"gh_available"`
	GHVersion    string `json:"gh_version,omitempty"`
	GHAuth       bool   `json:"gh_authenticated"`
}

// Doctor inspects system dependencies and CLI readiness.
func Doctor(ctx context.Context, p Params) result.Envelope {
	report := DoctorReport{}
	allGood := true
	var issues []string

	// Check git
	gitRes, err := p.Runner.Run(ctx, p.Dir, "git", "--version")
	if err == nil && gitRes.ExitCode == 0 {
		report.GitAvailable = true
		report.GitVersion = strings.TrimSpace(gitRes.Stdout)
		if ok, parsed := isVersionAtLeast(report.GitVersion, 2, 20); parsed && !ok {
			allGood = false
			issues = append(issues, fmt.Sprintf("git version is older than minimum required 2.20.0 (%s)", report.GitVersion))
		}
	} else {
		allGood = false
		issues = append(issues, "git binary not found or inaccessible")
	}

	// Check gh
	ghRes, err := p.Runner.Run(ctx, p.Dir, "gh", "--version")
	if err == nil && ghRes.ExitCode == 0 {
		report.GHAvailable = true
		lines := strings.Split(ghRes.Stdout, "\n")
		if len(lines) > 0 {
			report.GHVersion = strings.TrimSpace(lines[0])
		}
		if ok, parsed := isVersionAtLeast(report.GHVersion, 2, 0); parsed && !ok {
			allGood = false
			issues = append(issues, fmt.Sprintf("gh version is older than minimum required 2.0.0 (%s)", report.GHVersion))
		}

		// Check auth
		authRes, _ := p.Runner.Run(ctx, p.Dir, "gh", "auth", "status")
		if authRes.ExitCode == 0 {
			report.GHAuth = true
		} else {
			allGood = false
			issues = append(issues, "gh is not authenticated (run 'gh auth login')")
		}
	} else {
		allGood = false
		issues = append(issues, "gh CLI binary not found on PATH")
	}

	if !report.GitAvailable {
		return logAndBuildEnvelope(p, "doctor", result.NewEnvelope(
			result.ValidationFailed,
			"ENVIRONMENT_UNSUPPORTED",
			"DOCTOR",
			fmt.Sprintf("Environment check failed: %s", strings.Join(issues, "; ")),
			strPtr("Install git and ensure it is available in system PATH."),
			report,
			nil,
			nil,
		), false)
	}

	if !allGood {
		return logAndBuildEnvelope(p, "doctor", result.NewEnvelope(
			result.ActionRequired,
			"DIAGNOSTIC_WARNING",
			"DOCTOR",
			fmt.Sprintf("Doctor reported diagnostic warnings: %s", strings.Join(issues, "; ")),
			strPtr("Resolve GitHub CLI authentication or setup."),
			report,
			nil,
			nil,
		), false)
	}

	return logAndBuildEnvelope(p, "doctor", result.NewEnvelope(
		result.Success,
		"DOCTOR_OK",
		"DOCTOR",
		"All prerequisites (git, gh, authentication) are satisfied.",
		nil,
		report,
		nil,
		nil,
	), false)
}

func isVersionAtLeast(versionStr string, minMajor, minMinor int) (bool, bool) {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return false, false
	}
	major, err1 := strconv.Atoi(matches[1])
	minor, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return false, false
	}
	if major > minMajor {
		return true, true
	}
	if major == minMajor && minor >= minMinor {
		return true, true
	}
	return false, true
}