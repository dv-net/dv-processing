package workflow

import (
	"fmt"
)

// Snapshot represents a snapshot of the workflow.
type Snapshot struct {
	StepsStates   []*StepState  `json:"steps_states"`
	WorkflowState WorkflowState `json:"workflow_state"`
}

func GetArg[T any](sh Snapshot, stepName, key string) (T, error) {
	var arg T

	if stepName == "" {
		return arg, fmt.Errorf("step is empty")
	}

	if key == "" {
		return arg, fmt.Errorf("key is empty")
	}

	for _, state := range sh.StepsStates {
		if state.CurrentStep != stepName {
			continue
		}

		state.mu.RLock()
		args := state.Args
		state.mu.RUnlock()

		val, ok := args[key]
		if !ok {
			return arg, fmt.Errorf("value not found for key %s: %w", key, ErrNotFound)
		}

		arg, ok = val.(T)
		if !ok {
			return arg, fmt.Errorf("type assertion failed")
		}

		return arg, nil
	}

	return arg, fmt.Errorf("step not found")
}
