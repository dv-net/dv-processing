package workflow

import (
	"sync"
	"time"
)

/*

	StepState

*/

type StepState struct {
	PreviousStage *string `json:"previous_stage,omitempty"`
	PreviousStep  *string `json:"previous_step,omitempty"`
	CurrentStage  string  `json:"current_stage"`
	CurrentStep   string  `json:"current_step"`
	NextStage     *string `json:"next_stage,omitempty"`
	NextStep      *string `json:"next_step,omitempty"`

	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`

	Status StepStatus `json:"status"`

	Error string `json:"error,omitempty"`

	mu   sync.RWMutex
	Args map[string]any `json:"args,omitempty"`
}

func NewStepState() *StepState {
	return &StepState{}
}

// SetPreviousStage sets the previous stage for the step.
func (s *StepState) SetPreviousStage(stage string) *StepState {
	if stage != "" {
		s.PreviousStage = &stage
	}
	return s
}

// SetPreviousStep sets the previous step for the step.
func (s *StepState) SetPreviousStep(step string) *StepState {
	if step != "" {
		s.PreviousStep = &step
	}
	return s
}

// SetCurrentStage sets the current stage for the step.
func (s *StepState) SetCurrentStage(stage string) *StepState {
	s.CurrentStage = stage
	return s
}

// SetCurrentStep sets the current step for the step.
func (s *StepState) SetCurrentStep(step string) *StepState {
	s.CurrentStep = step
	return s
}

// SetNextStage sets the next stage for the step.
func (s *StepState) SetNextStage(stage string) *StepState {
	if stage != "" {
		s.NextStage = &stage
	}
	return s
}

// SetNextStep sets the next step for the step.
func (s *StepState) SetNextStep(step string) *StepState {
	if step != "" {
		s.NextStep = &step
	}
	return s
}

// SetStartTime sets the start time for the step.
func (s *StepState) SetStartTime(t time.Time) *StepState {
	if !t.IsZero() {
		s.StartTime = &t
	}
	return s
}

// SetEndTime sets the end time for the step.
func (s *StepState) SetEndTime(t time.Time) *StepState {
	if !t.IsZero() {
		s.EndTime = &t
	}
	return s
}

// SetStatus sets the status for the step.
func (s *StepState) SetStatus(status StepStatus) *StepState {
	s.Status = status
	return s
}

// SetError sets the error for the step.
func (s *StepState) SetError(err error) *StepState {
	if err == nil {
		s.Error = ""
	} else {
		s.Error = err.Error()
	}
	return s
}

// SetArgs sets the arguments for the step.
func (s *StepState) SetArgs(args map[string]any) (*StepState, error) {
	s.mu.Lock()
	s.Args = args
	s.mu.Unlock()

	return s, nil
}

// SetArgs sets the arguments for the step.
func (s *StepState) SetArg(key string, value any) (*StepState, error) {
	s.mu.Lock()
	if s.Args == nil {
		s.Args = make(map[string]any)
	}

	s.Args[key] = value
	s.mu.Unlock()

	return s, nil
}
