package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// Workflow represents the entire workflow composed of multiple stages.
type Workflow struct {
	// Name is the name of the workflow.
	Name string
	// State is the state of the workflow.
	State WorkflowState
	// Stages is the list of stages for the workflow.
	Stages []*Stage
	// BeforeFn is the before start function for the workflow.
	BeforeFn func(context.Context, *Workflow) error
	// AfterFn is the after complete function for the workflow.
	AfterFn func(context.Context, *Workflow) error
	// OnFailureFn is the on failure function for the workflow.
	OnFailureFn func(context.Context, *Workflow, error) error

	// BeforeAllStagesFn is the before all steps function for the workflow.
	BeforeAllStepsFn func(context.Context, *Workflow, *Stage, *Step) error
	// AfterAllStagesFn is the after all steps function for the workflow.
	AfterAllStepsFn func(context.Context, *Workflow, *Stage, *Step) error

	// Debug allows to enable debug mode.
	Debug bool

	logger logger

	isStopped atomic.Bool
	prevStep  *Step

	isSkipError bool
}

func New(opts ...WorkflowOption) *Workflow {
	wf := &Workflow{}

	for _, opt := range opts {
		opt(wf)
	}

	return wf
}

// Run executes the workflow.
func (w *Workflow) Run(ctx context.Context) (err error) {
	defer func() {
		if err == nil {
			return
		}

		if w.OnFailureFn != nil {
			failureErr := w.OnFailureFn(ctx, w, err)
			if failureErr != nil {
				if w.logger != nil {
					w.Debugf("workflow on failure: %s", failureErr)
				}
			}

			if w.isSkipError {
				err = nil
			}
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.execute(ctx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		w.isStopped.Store(true)
	}

	select {
	case <-time.After(9 * time.Second):
		return fmt.Errorf("workflow shutdown execution timeout")
	case err := <-errCh:
		return err
	}
}

func (w *Workflow) execute(ctx context.Context) (err error) {
	// initialize the workflow
	if err := w.init(); err != nil {
		return fmt.Errorf("workflow init: %w", err)
	}

	// check if workflow is already completed, suspended or failed
	if w.checkStatus() {
		return nil
	}

	if w.BeforeFn != nil {
		if err := w.BeforeFn(ctx, w); err != nil {
			return fmt.Errorf("workflow before: %w", err)
		}
	}

	var stepHandlerError error
	for _, stage := range w.Stages {
		if w.isStopped.Load() {
			return ctx.Err()
		}

		if err := w.handleStage(ctx, stage); err != nil {
			// skip stage
			if errors.Is(err, ErrSkipStage) {
				continue
			}

			// skip all stages
			if errors.Is(err, ErrBreakStages) {
				break
			}

			// exit from workflow
			if errors.Is(err, ErrExitWorkflow) {
				return nil
			}

			stepHandlerError = err

			break
		}
	}

	if stepHandlerError == nil {
		w.State.SetCompleted(true)
	}

	if w.AfterFn != nil {
		if err := w.AfterFn(ctx, w); err != nil {
			w.Errorf("workflow after: %s", err)
		}
	}

	return stepHandlerError
}

func (w *Workflow) init() error {
	// check steps in stages for unique names
	for stageIdx, stage := range w.Stages {
		if stage == nil {
			return fmt.Errorf("stage at index [%d] is nil", stageIdx)
		}

		uniqueSteps := make(map[string]struct{})
		for stepIdx, step := range stage.Steps {
			if step == nil {
				return fmt.Errorf("step at index [%d] in stage [%s] is nil", stepIdx, stage.Name)
			}

			if _, ok := uniqueSteps[step.Name]; ok {
				return fmt.Errorf("step [%s] is not unique in stage [%s]", step.Name, stage.Name)
			}
			uniqueSteps[step.Name] = struct{}{}

			if step.State == nil {
				step.State = NewStepState()
			}

			step.setDefaultValues()

			// set current stage and step
			step.State.SetCurrentStage(stage.Name)
			step.State.SetCurrentStep(step.Name)
		}
	}

	w.Debugf("start workflow: %s", w.Name)
	w.Debugf("stages count: %d", len(w.Stages))

	return nil
}

// handleStage handles the stage.
func (w *Workflow) handleStage(ctx context.Context, stage *Stage) error {
	nextStageName := w.State.NextStage
	if nextStageName != "" && nextStageName == stage.Name {
		w.State.SetNextStage("")
	}

	w.Debugf("start stage: %s", stage.Name)

	// check if workflow is already completed or suspended
	if w.checkStatus() {
		return ErrExitWorkflow
	}

	if nextStageName != "" {
		if nextStageName != stage.Name {
			w.Debugf("skipping stage: %s (next stage)", stage.Name)
			return ErrSkipStage
		}
		w.State.SetNextStage("")
	}

	if len(stage.Steps) == 0 {
		w.Warnf("skipping stage: %s (no steps)", stage.Name)
		return ErrSkipStage
	}

	if stage.BeforeFn != nil {
		if err := stage.BeforeFn(ctx, stage); err != nil {
			return err
		}
	}

	for _, step := range stage.Steps {
		if w.isStopped.Load() {
			return ctx.Err()
		}

		if err := w.handleStep(ctx, stage, step); err != nil {
			if errors.Is(err, ErrBreakStages) {
				return ErrBreakStages
			}

			if errors.Is(err, ErrSkipStep) {
				continue
			}

			if errors.Is(err, ErrExitWorkflow) {
				return ErrExitWorkflow
			}

			return err
		}
	}

	if stage.AfterFn != nil {
		if err := stage.AfterFn(ctx, stage); err != nil {
			return err
		}
	}

	return nil
}

// handleStep handles the step.
func (w *Workflow) handleStep(ctx context.Context, stage *Stage, step *Step) (err error) {
	nextStepName := w.State.NextStep
	if nextStepName != "" && nextStepName == step.Name {
		w.State.SetNextStep("")
	}

	w.Debugf("executing step: %s / %s", stage.Name, step.Name)

	// check if workflow is already completed or suspended
	if w.checkStatus() {
		return ErrExitWorkflow
	}

	if step.State.Status == StepStatusCompleted {
		w.Debugf("skipping step: %s (already completed)", step.Name)
		return ErrSkipStep
	}

	if step.State.Status == StepStatusSkipped {
		w.Debugf("skipping step: %s (skipped)", step.Name)
		return ErrSkipStep
	}

	if step.State.Status == StepStatusSuspended {
		w.Debugf("skipping step: %s (suspended)", step.Name)
		// Not sure if this is the correct return value
		return ErrBreakStages
	}

	if nextStepName != "" {
		if nextStepName != step.Name {
			w.Debugf("skipping step: %s (next step %s)", step.Name, nextStepName)
			step.State.SetStatus(StepStatusSkipped)
			return ErrSkipStep
		}
		w.State.SetNextStep("")
	}

	step.State.SetStartTime(time.Now())
	defer func() {
		step.State.SetEndTime(time.Now())
		if err == nil {
			step.State.SetError(nil)
			step.State.SetStatus(StepStatusCompleted)
		}

		if err != nil {
			if errors.Is(err, ErrSkipStep) || errors.Is(err, ErrBreakStages) || errors.Is(err, ErrBreakStages) {
				step.State.SetStatus(StepStatusSkipped)
			} else {
				step.State.SetStatus(StepStatusFailed)
				step.State.SetError(err)
			}
		}
	}()

	step.State.SetStatus(StepStatusPending)

	if w.prevStep != nil && w.prevStep.State != nil {
		step.State.SetPreviousStage(w.prevStep.State.CurrentStage)
		step.State.SetPreviousStep(w.prevStep.State.CurrentStep)
	}

	w.prevStep = step

	step.State.SetStatus(StepStatusProcessing)

	if step.Func == nil {
		return fmt.Errorf("step [%s] in stage [%s] has no function", step.Name, stage.Name)
	}

	if step.RetryPolicy == nil {
		return fmt.Errorf("step [%s] in stage [%s] has no retry policy", step.Name, stage.Name)
	}

	// run before all steps function
	if w.BeforeAllStepsFn != nil {
		if err := w.BeforeAllStepsFn(ctx, w, stage, step); err != nil {
			return err
		}
	}

	// run before function
	if step.BeforeFn != nil {
		if err := step.BeforeFn(ctx, step); err != nil {
			return err
		}
	}

	// run step function
	err = step.RetryPolicy(ctx, w, stage, step)

	// run after function
	if step.AfterFn != nil {
		if err := step.AfterFn(ctx, step); err != nil {
			return err
		}
	}

	// run after all steps function
	if w.AfterAllStepsFn != nil {
		if err := w.AfterAllStepsFn(ctx, w, stage, step); err != nil {
			return err
		}
	}

	if err != nil {
		return fmt.Errorf("stage [%s] failed on step [%s]: %w", stage.Name, step.Name, err)
	}

	if step.FinishWorkflow {
		return ErrBreakStages
	}

	return nil
}
