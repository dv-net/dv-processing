package workflow

import "context"

// WorkflowOption defines the function signature for a workflow option.
type WorkflowOption func(*Workflow) //nolint:revive

// WithName sets the name for the workflow.
func WithName(n string) WorkflowOption {
	return func(w *Workflow) {
		w.Name = n
	}
}

// WithDebug sets the debug flag for the workflow.
func WithDebug(d bool) WorkflowOption {
	return func(w *Workflow) {
		w.Debug = d
	}
}

// WithState sets the state for the workflow.
func WithState(s WorkflowState) WorkflowOption {
	return func(w *Workflow) {
		w.State = s
	}
}

// WithLogger sets the logger for the workflow.
func WithLogger(l logger) WorkflowOption {
	return func(w *Workflow) {
		w.logger = l
	}
}

// WithBeforeFn sets the before start function for the workflow.
func WithBeforeFn(fn func(context.Context, *Workflow) error) WorkflowOption {
	return func(w *Workflow) {
		w.BeforeFn = fn
	}
}

// WithAfterFn sets the after complete function for the workflow.
func WithAfterFn(fn func(context.Context, *Workflow) error) WorkflowOption {
	return func(w *Workflow) {
		w.AfterFn = fn
	}
}

// WithBeforeAllStepsFn sets the before all steps function for the workflow.
func WithBeforeAllStepsFn(fn func(context.Context, *Workflow, *Stage, *Step) error) WorkflowOption {
	return func(w *Workflow) {
		w.BeforeAllStepsFn = fn
	}
}

// WithAfterAllStepsFn sets the after all steps function for the workflow.
func WithAfterAllStepsFn(fn func(context.Context, *Workflow, *Stage, *Step) error) WorkflowOption {
	return func(w *Workflow) {
		w.AfterAllStepsFn = fn
	}
}

// WithOnFailureFn sets the on failure function for the workflow.
func WithOnFailureFn(fn func(context.Context, *Workflow, error) error) WorkflowOption {
	return func(w *Workflow) {
		w.OnFailureFn = fn
	}
}
