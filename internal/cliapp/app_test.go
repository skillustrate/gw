package cliapp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/result"
)

func TestApp_InspectJSON(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel":      {ExitCode: 0, Stdout: "/test/repo\n"},
		"git rev-parse --git-dir":            {ExitCode: 0, Stdout: "/test/repo/.git\n"},
		"git status --porcelain=v2 --branch": {ExitCode: 0, Stdout: "# branch.head main\n"},
		"gh auth status":                     {ExitCode: 0, Stdout: "Logged in\n"},
		"gh repo view --json nameWithOwner":  {ExitCode: 0, Stdout: `{"nameWithOwner":"org/repo"}`},
		"gh pr view main --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 1,
			Stderr:   "no PR found",
		},
	})

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, fake)

	exitCode := app.Run(context.Background(), []string{"inspect", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	var env result.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v\nOutput was:\n%s", err, stdout.String())
	}

	if env.Status != result.Success || env.Code != "OK" {
		t.Errorf("expected SUCCESS(OK), got %+v", env)
	}
}

func TestApp_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, exec.NewFakeRunner(nil))

	exitCode := app.Run(context.Background(), []string{"invalid-cmd", "--json"})
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for usage error, got %d", exitCode)
	}

	var env result.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if env.Status != result.ValidationFailed || env.Code != "USAGE_ERROR" {
		t.Errorf("expected VALIDATION_FAILED(USAGE_ERROR), got %+v", env)
	}
}

func TestApp_HumanReadableOutput(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git --version":  {ExitCode: 0, Stdout: "git version 2.44.0\n"},
		"gh --version":   {ExitCode: 0, Stdout: "gh version 2.45.0\n"},
		"gh auth status": {ExitCode: 0, Stdout: "Logged in\n"},
	})

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, fake)

	exitCode := app.Run(context.Background(), []string{"doctor"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "[SUCCESS]") || !strings.Contains(outStr, "DOCTOR_OK") {
		t.Errorf("expected human readable status line, got:\n%s", outStr)
	}
}

func TestApp_PolicyFlags(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel":      {ExitCode: 0, Stdout: "/test/repo\n"},
		"git rev-parse --git-dir":            {ExitCode: 0, Stdout: "/test/repo/.git\n"},
		"git status --porcelain=v2 --branch": {ExitCode: 0, Stdout: "# branch.head main\n"},
		"gh auth status":                     {ExitCode: 0, Stdout: "Logged in\n"},
		"gh repo view --json nameWithOwner":  {ExitCode: 0, Stdout: `{"nameWithOwner":"org/repo"}`},
		"gh pr view main --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 1,
			Stderr:   "no PR found",
		},
	})

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, fake)

	// Test passing policy override flags
	exitCode := app.Run(context.Background(), []string{
		"inspect",
		"--json",
		"--commit-allow",
		"--push-allow",
		"--push-set-upstream",
		"--merge-allow",
		"--merge-method", "rebase",
	})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
}

func TestApp_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, exec.NewFakeRunner(nil))

	exitCode := app.Run(context.Background(), []string{"help"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	helpOut := stdout.String()
	if !strings.Contains(helpOut, "--commit-allow") || !strings.Contains(helpOut, "--push-allow") {
		t.Errorf("expected help output to document policy override flags, got:\n%s", helpOut)
	}
}

func TestApp_PolicyFlagsExplicitBoolValues(t *testing.T) {
	fake := exec.NewFakeRunner(map[string]exec.Result{
		"git rev-parse --show-toplevel":      {ExitCode: 0, Stdout: "/test/repo\n"},
		"git rev-parse --git-dir":            {ExitCode: 0, Stdout: "/test/repo/.git\n"},
		"git status --porcelain=v2 --branch": {ExitCode: 0, Stdout: "# branch.head main\n"},
		"gh auth status":                     {ExitCode: 0, Stdout: "Logged in\n"},
		"gh repo view --json nameWithOwner":  {ExitCode: 0, Stdout: `{"nameWithOwner":"org/repo"}`},
		"gh pr view main --json number,url,state,isDraft,mergeable,statusCheckRollup,reviewDecision": {
			ExitCode: 1,
			Stderr:   "no PR found",
		},
	})

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, fake)

	// Test passing boolean flags with explicit space-separated true/false values
	exitCode := app.Run(context.Background(), []string{
		"inspect",
		"--json",
		"--commit-allow", "true",
		"--push-allow", "false",
		"--push-set-upstream", "true",
		"--merge-allow", "false",
	})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
}