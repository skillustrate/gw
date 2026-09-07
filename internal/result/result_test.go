package result

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestEnvelope_ExitCodeTable(t *testing.T) {
	tests := []struct {
		name     string
		envelope Envelope
		wantCode int
	}{
		{"SUCCESS", Envelope{Status: Success}, 0},
		{"NOOP", Envelope{Status: Noop}, 0},
		{"WAIT", Envelope{Status: Wait}, 0},
		{"ACTION_REQUIRED", Envelope{Status: ActionRequired}, 1},
		{"BLOCKED", Envelope{Status: Blocked}, 1},
		{"AUTH_REQUIRED", Envelope{Status: AuthRequired}, 1},
		{"CONFLICT", Envelope{Status: Conflict}, 1},
		{"POLICY_DENIED", Envelope{Status: PolicyDenied}, 1},
		{"VALIDATION_FAILED (usage)", Envelope{Status: ValidationFailed, Code: "USAGE_ERROR"}, 2},
		{"VALIDATION_FAILED (flag)", Envelope{Status: ValidationFailed, Code: "INVALID_FLAG"}, 2},
		{"VALIDATION_FAILED (env)", Envelope{Status: ValidationFailed, Code: "ENVIRONMENT_UNSUPPORTED"}, 1},
		{"COMMAND_FAILED", Envelope{Status: CommandFailed}, 1},
		{"VERIFICATION_FAILED", Envelope{Status: VerificationFailed}, 1},
		{"HUMAN_APPROVAL_REQUIRED", Envelope{Status: HumanApprovalRequired}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.envelope.ExitCode()
			if got != tt.wantCode {
				t.Errorf("ExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestEnvelope_GoldenFiles(t *testing.T) {
	cases := map[string]Envelope{
		"success.json": {
			SchemaVersion: 1,
			Status:        Success,
			Code:          "PR_CREATED",
			Action:        "CREATE_PR",
			Reason:        "Working tree clean, branch pushed, no existing PR.",
			NextAction:    nil,
			Data: map[string]any{
				"pr": map[string]any{
					"number": float64(142),
					"url":    "https://github.com/owner/repo/pull/142",
				},
			},
		},
		"noop.json": {
			SchemaVersion: 1,
			Status:        Noop,
			Code:          "ALREADY_PUSHED",
			Action:        "PUSH",
			Reason:        "Branch is already synchronized with remote upstream.",
			NextAction:    nil,
		},
		"action_required.json": {
			SchemaVersion: 1,
			Status:        ActionRequired,
			Code:          "PUSH_REQUIRED",
			Action:        "PR_CREATE",
			Reason:        "Branch has unpushed commits.",
			NextAction:    strPtr("gw push --json"),
		},
		"wait.json": {
			SchemaVersion:  1,
			Status:         Wait,
			Code:           "CHECKS_PENDING",
			Action:         "PR_STATUS",
			Reason:         "CI checks are still running.",
			NextAction:     strPtr("gw pr-status --json"),
			RetryAfterHint: intPtr(30),
		},
		"blocked.json": {
			SchemaVersion: 1,
			Status:        Blocked,
			Code:          "WORKTREE_DIRTY",
			Action:        "PUSH",
			Reason:        "Working tree contains uncommitted changes.",
			NextAction:    strPtr("gw prepare --json"),
		},
		"auth_required.json": {
			SchemaVersion: 1,
			Status:        AuthRequired,
			Code:          "GH_UNAUTHENTICATED",
			Action:        "PR_CREATE",
			Reason:        "GitHub CLI is not authenticated.",
			NextAction:    strPtr("gh auth login"),
		},
		"conflict.json": {
			SchemaVersion: 1,
			Status:        Conflict,
			Code:          "BRANCH_DIVERGED",
			Action:        "SYNC",
			Reason:        "Local branch has diverged from upstream tracking branch.",
			NextAction:    strPtr("Resolve divergence manually or rebase before syncing."),
		},
		"policy_denied.json": {
			SchemaVersion: 1,
			Status:        PolicyDenied,
			Code:          "PUSH_DISALLOWED",
			Action:        "PUSH",
			Reason:        "Policy disallows pushing to protected base branch directly.",
			NextAction:    nil,
		},
		"validation_failed.json": {
			SchemaVersion: 1,
			Status:        ValidationFailed,
			Code:          "USAGE_ERROR",
			Action:        "UNKNOWN",
			Reason:        "Unrecognized command flag: --unknown",
			NextAction:    strPtr("Run gw --help for valid usage."),
		},
		"command_failed.json": {
			SchemaVersion: 1,
			Status:        CommandFailed,
			Code:          "GIT_EXEC_ERROR",
			Action:        "FETCH",
			Reason:        "Git command failed to connect to remote repository.",
			NextAction:    strPtr("Check network connection and retry."),
		},
		"verification_failed.json": {
			SchemaVersion: 1,
			Status:        VerificationFailed,
			Code:          "POST_MUTATION_MISMATCH",
			Action:        "PUSH",
			Reason:        "Branch verification failed after push execution.",
			NextAction:    strPtr("gw inspect --json"),
		},
		"human_approval_required.json": {
			SchemaVersion: 1,
			Status:        HumanApprovalRequired,
			Code:          "REVIEW_APPROVAL_REQUIRED",
			Action:        "PR_MERGE",
			Reason:        "PR requires human review approval on GitHub before merging.",
			NextAction:    strPtr("Obtain reviewer approval on GitHub PR."),
		},
	}

	for filename, expectedEnv := range cases {
		t.Run(filename, func(t *testing.T) {
			path := filepath.Join("testdata", "golden", filename)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", path, err)
			}

			// Validate JSON parses back identically into Envelope
			var parsed Envelope
			if err := json.Unmarshal(content, &parsed); err != nil {
				t.Fatalf("failed to unmarshal golden file %s: %v", filename, err)
			}

			if parsed.Status != expectedEnv.Status {
				t.Errorf("%s: Status = %v, want %v", filename, parsed.Status, expectedEnv.Status)
			}
			if parsed.Code != expectedEnv.Code {
				t.Errorf("%s: Code = %v, want %v", filename, parsed.Code, expectedEnv.Code)
			}
			if parsed.Action != expectedEnv.Action {
				t.Errorf("%s: Action = %v, want %v", filename, parsed.Action, expectedEnv.Action)
			}
			if parsed.Reason != expectedEnv.Reason {
				t.Errorf("%s: Reason = %v, want %v", filename, parsed.Reason, expectedEnv.Reason)
			}
			if (parsed.NextAction == nil && expectedEnv.NextAction != nil) ||
				(parsed.NextAction != nil && expectedEnv.NextAction == nil) ||
				(parsed.NextAction != nil && expectedEnv.NextAction != nil && *parsed.NextAction != *expectedEnv.NextAction) {
				t.Errorf("%s: NextAction mismatch", filename)
			}

			// Serialize and ensure no errors
			serialized, err := expectedEnv.ToJSON()
			if err != nil {
				t.Fatalf("failed to serialize: %v", err)
			}
			if !strings.Contains(string(serialized), string(expectedEnv.Status)) {
				t.Errorf("serialized JSON missing status %v", expectedEnv.Status)
			}
		})
	}
}