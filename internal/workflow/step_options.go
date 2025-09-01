package workflow

import (
	"context"
	"time"
)

// StepOption defines the function signature for a step option.
type StepOption func(*Step)

// WithStepBeforeFn sets the before start function for the step.
func WithStepBeforeFn(fn func(context.Context, *Step) error) StepOption {
	return func(s *Step) {
		s.BeforeFn = fn
	}
}

// WithStepAfterFn sets the after complete function for the step.
func WithStepAfterFn(fn func(context.Context, *Step) error) StepOption {
	return func(s *Step) {
		s.AfterFn = fn
	}
}

// WithStepTimeout sets the timeout for the step.
func WithStepTimeout(d time.Duration) StepOption {
	return func(s *Step) {
		s.Timeout = d
	}
}

// WithStepRetryPolicy sets the retry policy for the step.
func WithStepRetryPolicy(fn RetryPolicyFn) StepOption {
	return func(s *Step) {
		s.RetryPolicy = fn
	}
}

// WithStepMaxRetries sets the max retries for the step.
func WithStepMaxRetries(r int) StepOption {
	return func(s *Step) {
		s.MaxRetries = r
	}
}

// WithStepArgs sets the arguments for the step.
func WithStepArgs(args any) StepOption {
	return func(s *Step) {
		s.Args = args
	}
}

// WithStepKind sets the kind for the step.
func WithStepKind(k string) StepOption {
	return func(s *Step) {
		s.Kind = k
	}
}
