package workflow

import (
	"context"
	"encoding/json"
	"slices"
)

/*

	Setters for the Workflow struct

*/

// SetName sets the name for the workflow.
func (w *Workflow) SetName(name string) *Workflow {
	w.Name = name
	return w
}

// SetState sets the state for the workflow.
func (w *Workflow) SetState(state WorkflowState) *Workflow {
	w.State = state
	return w
}

// SetStages sets the stages for the workflow.
func (w *Workflow) SetStages(stages []*Stage) *Workflow {
	w.Stages = stages
	return w
}

// SetLogger sets the logger for the workflow.
func (w *Workflow) SetLogger(l logger) *Workflow {
	w.logger = l
	return w
}

// SetDebug sets the debug flag for the workflow.
func (w *Workflow) SetDebug(debug bool) *Workflow {
	w.Debug = debug
	return w
}

// SetBeforeFn sets the before start function for the workflow.
func (w *Workflow) SetBeforeFn(fn func(context.Context, *Workflow) error) *Workflow {
	w.BeforeFn = fn
	return w
}

// SetBeforeAllStepsFn sets the before start function for the workflow.
func (w *Workflow) SetBeforeAllStepsFn(fn func(context.Context, *Workflow, *Stage, *Step) error) *Workflow {
	w.BeforeAllStepsFn = fn
	return w
}

// SetAfterFn sets the after complete function for the workflow.
func (w *Workflow) SetAfterFn(fn func(context.Context, *Workflow) error) *Workflow {
	w.AfterFn = fn
	return w
}

// SetOnFailureFn sets the on failure function for the workflow.
func (w *Workflow) SetOnFailureFn(fn func(context.Context, *Workflow, error) error) *Workflow {
	w.OnFailureFn = fn
	return w
}

// SetSkipError
func (w *Workflow) SetSkipError(skip bool) *Workflow {
	w.isSkipError = skip
	return w
}

/*

	Getters for the Workflow struct

*/

// Logger returns the logger for the workflow.
func (w *Workflow) Logger() logger { return w.logger }

// Step states
func (w *Workflow) StepStates() []*StepState {
	states := []*StepState{}
	for _, stage := range w.Stages {
		for _, step := range stage.Steps {
			if step.State.Status.Valid() {
				states = append(states, step.State)
			}
		}
	}
	return states
}

// GetSnapshot
func (w *Workflow) GetSnapshot() Snapshot {
	return Snapshot{
		StepsStates:   w.StepStates(),
		WorkflowState: w.State,
	}
}

// GetSnapshot
func (w *Workflow) GetJSONSnapshot() string {
	sh := Snapshot{
		StepsStates:   w.StepStates(),
		WorkflowState: w.State,
	}

	var b []byte
	if w.Debug {
		b, _ = json.MarshalIndent(sh, "", "  ")
	} else {
		b, _ = json.Marshal(sh)
	}

	return string(b)
}

// SetSnapshot
func (w *Workflow) SetSnapshot(snapshot Snapshot) error {
	for _, state := range snapshot.StepsStates {
		for _, stage := range w.Stages {
			for _, step := range stage.Steps {
				if stage.Name == state.CurrentStage &&
					step.Name == state.CurrentStep {
					step.State = state
				}
			}
		}
	}

	w.State = snapshot.WorkflowState

	return nil
}

// SetSnapshot
func (w *Workflow) SetJSONSnapshot(snapshot string) error {
	sh := Snapshot{}
	err := json.Unmarshal([]byte(snapshot), &sh)
	if err != nil {
		return err
	}

	for _, state := range sh.StepsStates {
		for _, stage := range w.Stages {
			for _, step := range stage.Steps {
				if stage.Name == state.CurrentStage &&
					step.Name == state.CurrentStep {
					step.State = state
				}
			}
		}
	}

	w.State = sh.WorkflowState

	return nil
}

// CurrentStage returns the current stage for the workflow.
func (w *Workflow) CurrentStage() *Stage {
	var res *Stage
	if len(w.Stages) > 0 {
		res = w.Stages[0]
	}

	for _, stage := range slices.Backward(w.Stages) {
		for _, step := range slices.Backward(stage.Steps) {
			if step.State.Status.Valid() {
				return stage
			}
		}
	}

	return res
}

// CurrentStep returns the current step for the workflow.
func (w *Workflow) CurrentStep() *Step {
	currentStage := w.CurrentStage().Steps

	var res *Step
	if len(currentStage) > 0 {
		res = currentStage[0]
	}

	for _, step := range slices.Backward(currentStage) {
		if step.State.Status.Valid() {
			return step
		}
	}

	return res
}
