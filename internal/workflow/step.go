package workflow

import (
	"context"
	"time"
)

// StepFunc defines the function signature for a step.
type StepFunc func(ctx context.Context, workflow *Workflow, stage *Stage, step *Step) error

// Step represents a unique step within a stage.
type Step struct {
	// Name is the name of the step. Name should be unique within a stage.
	Name string
	// Kind is the kind of the step. Based on the step kind you can implement similar triggers in the workflow.
	Kind string
	// Func is the function to be executed for the step.
	Func StepFunc
	// Args is the arguments for the step.
	Args any
	// Timeout is the duration for the step. By default, it is set to 1 second.
	Timeout time.Duration
	// MaxRetries is the maximum number of retries for the step. By default, it is set to 1.
	MaxRetries int
	// RetryPolicy is the retry policy for the step. By default, it is set by timeout.
	RetryPolicy RetryPolicyFn
	// BeforeFn is the before start function for the step.
	BeforeFn func(context.Context, *Step) error
	// AfterFn is the after complete function for the step.
	AfterFn func(context.Context, *Step) error

	// State
	State *StepState

	// FinishWorkflow is a flag to finish the workflow after the step.
	FinishWorkflow bool
}

func (s *Step) setDefaultValues() {
	if s.Timeout == 0 {
		s.Timeout = 1 * time.Second
	}

	if s.MaxRetries == 0 {
		s.MaxRetries = 1
	}

	if s.RetryPolicy == nil {
		s.RetryPolicy = RunWithLinear
	}
}

// NewStep returns a new step.
func NewStep(name string, fn StepFunc, opts ...StepOption) *Step {
	s := &Step{
		Name: name,
		Func: fn,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
