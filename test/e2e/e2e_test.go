//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gitskill/gw/internal/cache"
	"github.com/gitskill/gw/internal/exec"
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
	"github.com/gitskill/gw/internal/workflow"
)

func TestE2E_FullLifecycle(t *testing.T) {
	e2eRepo := os.Getenv("GW_E2E_REPO")
	if e2eRepo == "" {
		t.Skip("GW_E2E_REPO environment variable not set; skipping live GitHub e2e test")
	}

	runner := exec.RealRunner{Timeout: 60 * time.Second}
	ctx := context.Background()

	pol := policy.Default()
	pol.Merge.Allow = true
	pol.Approved = true

	p := workflow.Params{
		Dir:           e2eRepo,
		Runner:        runner,
		Policy:        pol,
		EngineVersion: "0.1.0",
		Cache:         cache.New[state.RepoState](0),
	}

	// Step 1: Doctor
	docEnv := workflow.Doctor(ctx, p)
	t.Logf("Doctor report: %+v", docEnv)

	// Step 2: Inspect
	env := workflow.Inspect(ctx, p)
	if env.Status != result.Success {
		t.Fatalf("Inspect failed in e2e repo: %+v", env)
	}

	// Step 3: PR Ready
	readyEnv := workflow.PRReady(ctx, p)
	t.Logf("Observed PR ready status: %+v", readyEnv)

	// Step 4: PR Status
	statusEnv := workflow.PRStatus(ctx, p)
	t.Logf("Observed PR status: %+v", statusEnv)

	// Step 5: Prepare (read-only/validate branch setup)
	prepEnv := workflow.Prepare(ctx, p)
	t.Logf("Prepare status: %+v", prepEnv)
}