package workflow

import (
	"errors"
	"fmt"
)

var (
	// ErrSkipStep is used to skip the current step.
	ErrSkipStep = errors.New("skip step")
	// ErrSkipStage is used to skip the current step.
	ErrSkipStage = errors.New("skip stage")
	// ErrBreakStages is used to break the root loop.
	ErrBreakStages = errors.New("break stages")
	// ErrExitWorkflow is used to exit the workflow.
	ErrExitWorkflow = errors.New("exit")
	// ErrNoConsole
	ErrNoConsole = errors.New("no console")
	// NotFound
	ErrNotFound = errors.New("not found")
)

func NoConsoleError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%w: %w", ErrNoConsole, err)
}
