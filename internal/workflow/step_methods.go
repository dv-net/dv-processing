package workflow

import (
	"context"
	"time"
)

// SetFunc sets the step function.
func (s *Step) SetFunc(value StepFunc) *Step {
	s.Func = value
	return s
}

// SetTimeout sets the timeout.
func (s *Step) SetTimeout(value time.Duration) *Step {
	s.Timeout = value
	return s
}

// SetMaxRetries sets the max retries.
func (s *Step) SetMaxRetries(value int) *Step {
	s.MaxRetries = value
	return s
}

// SetRetryPolicy sets the retry policy.
func (s *Step) SetRetryPolicy(value RetryPolicyFn) *Step {
	s.RetryPolicy = value
	return s
}

// SetBeforeFn sets the before function.
func (s *Step) SetBeforeFn(value func(context.Context, *Step) error) *Step {
	s.BeforeFn = value
	return s
}

// SetAfterFn sets the after function.
func (s *Step) SetAfterFn(value func(context.Context, *Step) error) *Step {
	s.AfterFn = value
	return s
}

// SetStepFinishWorkflow sets the finish workflow flag.
func (s *Step) SetStepFinishWorkflow(value bool) *Step {
	s.FinishWorkflow = value
	return s
}
