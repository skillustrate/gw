package decision

import (
	"github.com/gitskill/gw/internal/policy"
	"github.com/gitskill/gw/internal/result"
	"github.com/gitskill/gw/internal/state"
)

// InspectDecide evaluates the state for gw inspect.
func InspectDecide(s state.RepoState, p policy.Policy) Decision {
	if !s.Repository.IsGitRepo {
		return Decision{
			Action: "INSPECT",
			Status: result.Success,
			Code:   "NOT_A_REPO",
			Reason: "Directory is not a Git repository.",
		}
	}

	return Decision{
		Action: "INSPECT",
		Status: result.Success,
		Code:   "OK",
		Reason: "Repository state observed successfully.",
	}
}

var Inspect Decider = InspectDecide