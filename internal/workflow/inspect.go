package workflow

import (
	"context"

	"github.com/gitskill/gw/internal/decision"
	"github.com/gitskill/gw/internal/redact"
	"github.com/gitskill/gw/internal/result"
)

// Inspect executes the read-only observe workflow.
func Inspect(ctx context.Context, p Params) result.Envelope {
	st, err := observeState(ctx, p, "")
	if err != nil {
		return logAndBuildEnvelope(p, "inspect", result.NewEnvelope(
			result.CommandFailed,
			"OBSERVE_FAILED",
			"INSPECT",
			"Failed to observe repository state: "+redact.Scrub(err.Error()),
			nil,
			nil,
			nil,
			nil,
		), false)
	}

	d := decision.Inspect(st, p.Policy)
	return logAndBuildEnvelope(p, "inspect", result.NewEnvelope(
		d.Status,
		d.Code,
		d.Action,
		d.Reason,
		d.NextAction,
		st,
		nil,
		nil,
	), false)
}