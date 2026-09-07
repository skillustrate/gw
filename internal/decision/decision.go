package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// Decision captures the pure logical determination of a workflow decider.
type Decision struct {
	Action     string        `json:"action"`
	Status     result.Status `json:"status"`
	Code       string        `json:"code"`
	Reason     string        `json:"reason"`
	NextAction *string       `json:"next_action"`
}

// Decider evaluates normalized RepoState and Policy into a pure Decision.
type Decider func(s state.RepoState, p policy.Policy) Decision

func strPtr(s string) *string { return &s }