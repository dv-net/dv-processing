package workflow

import "context"

// StageOption defines the function signature for a stage option.
type StageOption func(*Stage)

// WithStageBeforeFn sets the before start function for the stage.
func WithStageBeforeFn(fn func(context.Context, *Stage) error) StageOption {
	return func(s *Stage) {
		s.BeforeFn = fn
	}
}

// WithStageAfterFn sets the after complete function for the stage.
func WithStageAfterFn(fn func(context.Context, *Stage) error) StageOption {
	return func(s *Stage) {
		s.AfterFn = fn
	}
}

// WithStageSteps sets the steps for the stage.
func WithStageSteps(steps []*Step) StageOption {
	return func(s *Stage) {
		s.Steps = steps
	}
}
