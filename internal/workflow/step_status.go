package workflow

type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusProcessing StepStatus = "processing"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusSuspended  StepStatus = "suspended"
	StepStatusSkipped    StepStatus = "skipped"
)

// String
func (s StepStatus) String() string { return string(s) }

// Valid
func (s StepStatus) Valid() bool {
	switch s {
	case StepStatusPending,
		StepStatusProcessing,
		StepStatusCompleted,
		StepStatusFailed,
		StepStatusSuspended,
		StepStatusSkipped:
		return true
	}
	return false
}

// AllStepStatuses
func AllStepStatuses() []StepStatus {
	return []StepStatus{
		StepStatusPending,
		StepStatusProcessing,
		StepStatusCompleted,
		StepStatusFailed,
		StepStatusSuspended,
		StepStatusSkipped,
	}
}
