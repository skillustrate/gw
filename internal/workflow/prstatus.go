package workflow

import (
	"context"
	"strconv"

	"github.com/gitskill/gw/internal/decision"
	"github.com/gitskill/gw/internal/redact"
	"github.com/gitskill/gw/internal/result"
)

// PRStatus retrieves the normalized pull request status.
func PRStatus(ctx context.Context, p Params) result.Envelope {
	branchOverride := ""
	if p.PRNumber > 0 {
		branchOverride = strconv.Itoa(p.PRNumber)
	}

	st, err := observeState(ctx, p, branchOverride)
	if err != nil {
		return logAndBuildEnvelope(p, "pr-status", result.NewEnvelope(
			result.CommandFailed, "OBSERVE_FAILED", "PR_STATUS", "Failed to observe repository state: "+redact.Scrub(err.Error()), nil, nil, nil, nil,
		), false)
	}

	d := decision.PRStatus(st, p.Policy)
	return logAndBuildEnvelope(p, "pr-status", result.NewEnvelope(
		d.Status, d.Code, d.Action, d.Reason, d.NextAction, st, nil, nil,
	), false)
}