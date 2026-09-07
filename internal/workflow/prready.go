package workflow

import (
	"context"

	"github.com/gitskill/gw/internal/decision"
	"github.com/gitskill/gw/internal/redact"
	"github.com/gitskill/gw/internal/result"
)

// PRReady executes the read-only PR readiness evaluation workflow.
func PRReady(ctx context.Context, p Params) result.Envelope {
	st, err := observeState(ctx, p, "")
	if err != nil {
		return logAndBuildEnvelope(p, "pr-ready", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "PR_READY", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}

	d := decision.PRReady(st, p.Policy)
	return logAndBuildEnvelope(p, "pr-ready", result.NewEnvelope(
		d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, nil, nil,
	), false)
}