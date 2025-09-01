package workflow

import "encoding/json"

// WorkflowState represents the state of the workflow.
type WorkflowState struct { //nolint:revive
	// IsSuspended is the flag to indicate if the workflow is suspended (in pause).
	IsSuspended bool `json:"is_suspended"`
	// IsCompleted is the flag to indicate if the workflow is fully completed.
	IsCompleted bool `json:"is_completed"`
	// IsFailed is the flag to indicate if the workflow is failed.
	IsFailed bool `json:"is_failed"`

	// CustomError information about fail
	CustomError json.RawMessage `json:"custom_error,omitempty"`

	// Error formatted error msg
	Error string `json:"error,omitempty"`

	// NextStage is the next stage for the workflow.
	NextStage string `json:"next_stage,omitempty"`
	// NextStep is the next step for the workflow.
	NextStep string `json:"next_step,omitempty"`
}

type FailData struct {
	FailedStepName  string `json:"failed_step_name,omitempty"`
	FailedStageName string `json:"failed_stage_name,omitempty"`
}

// SetSuspended sets the suspended flag for the workflow.
func (w *WorkflowState) SetSuspended(value bool) *WorkflowState {
	w.IsSuspended = value
	return w
}

// SetCompleted sets the completed flag for the workflow.
func (w *WorkflowState) SetCompleted(value bool) *WorkflowState {
	w.IsCompleted = value
	return w
}

// SetFailed sets the failed flag for the workflow.
func (w *WorkflowState) SetFailed(value bool) *WorkflowState {
	w.IsFailed = value
	return w
}

func (w *WorkflowState) SetCustomError(value []byte) *WorkflowState {
	w.CustomError = value
	return w
}

func (w *WorkflowState) GetCustomError() json.RawMessage {
	return w.CustomError
}

func (w *WorkflowState) SetErrorMsg(msg string) *WorkflowState {
	w.Error = msg
	return w
}

// SetError sets the error for the workflow.
func (w *WorkflowState) SetError(err error) *WorkflowState {
	if err == nil {
		return w
	}

	w.Error = err.Error()
	return w
}

// SetNextStage sets the next stage for the workflow.
func (w *WorkflowState) SetNextStage(stage string) *WorkflowState {
	w.NextStage = stage
	return w
}

// SetNextStep sets the next step for the workflow.
func (w *WorkflowState) SetNextStep(step string) *WorkflowState {
	w.NextStep = step
	return w
}
