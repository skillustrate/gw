package result

import (
	"encoding/json"
)

// CurrentSchemaVersion is the current JSON envelope schema version.
const CurrentSchemaVersion = 1

// Status defines machine-readable workflow outcome categories.
type Status string

const (
	Success               Status = "SUCCESS"
	Noop                  Status = "NOOP"
	ActionRequired        Status = "ACTION_REQUIRED"
	Wait                  Status = "WAIT"
	Blocked               Status = "BLOCKED"
	AuthRequired          Status = "AUTH_REQUIRED"
	Conflict              Status = "CONFLICT"
	PolicyDenied          Status = "POLICY_DENIED"
	ValidationFailed      Status = "VALIDATION_FAILED"
	CommandFailed         Status = "COMMAND_FAILED"
	VerificationFailed    Status = "VERIFICATION_FAILED"
	HumanApprovalRequired Status = "HUMAN_APPROVAL_REQUIRED"
)

// Envelope is the unified JSON response contract returned to AI coding harnesses and CLI callers.
type Envelope struct {
	SchemaVersion  int     `json:"schema_version"`
	Status         Status  `json:"status"`
	Code           string  `json:"code"`
	Action         string  `json:"action"`
	Reason         string  `json:"reason"`
	NextAction     *string `json:"next_action"`
	Data           any     `json:"data,omitempty"`
	RetryAfterHint *int    `json:"retry_after_hint"`
	Debug          *string `json:"debug,omitempty"`
}

// ExitCode computes the process exit code according to the §15 error taxonomy:
// 0: SUCCESS, NOOP, WAIT
// 2: Usage errors (flags/CLI syntax)
// 1: Caller-actionable failures / blocked conditions
func (e Envelope) ExitCode() int {
	switch e.Status {
	case Success, Noop, Wait:
		return 0
	case ValidationFailed:
		if e.Code == "USAGE_ERROR" || e.Code == "INVALID_FLAG" {
			return 2
		}
		return 1
	default:
		return 1
	}
}

// ToJSON formats the envelope into indented JSON with a trailing newline.
func (e Envelope) ToJSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// NewEnvelope creates a populated Envelope with the default current schema version.
func NewEnvelope(status Status, code, action, reason string, nextAction *string, data any, retryAfter *int, debug *string) Envelope {
	return Envelope{
		SchemaVersion:  CurrentSchemaVersion,
		Status:         status,
		Code:           code,
		Action:         action,
		Reason:         reason,
		NextAction:     nextAction,
		Data:           data,
		RetryAfterHint: retryAfter,
		Debug:          debug,
	}
}