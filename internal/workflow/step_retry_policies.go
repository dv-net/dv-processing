package workflow

import (
	"context"
	"errors"
	"time"
)

type RetryPolicyFn func(context.Context, *Workflow, *Stage, *Step) error

// RunWithLinear runs a step function with a simple linear retry mechanism.
func RunWithLinear(ctx context.Context, w *Workflow, stage *Stage, step *Step) error {
	var err error

	for retries := 0; retries < step.MaxRetries || step.MaxRetries < 0; retries++ {
		err = step.Func(ctx, w, stage, step)
		if err == nil {
			return nil
		}

		if !errors.Is(err, ErrNoConsole) {
			w.Errorf("step [%s] failed with error: %s", step.Name, err)
		}

		if retries+1 < step.MaxRetries {
			w.Errorf("retrying in %s", step.Timeout)
			time.Sleep(step.Timeout)
		}
	}

	return err
}

// RunWithBackoff runs a step function with a simple backoff retry mechanism.
func RunWithBackoff(ctx context.Context, w *Workflow, stage *Stage, step *Step) error {
	var err error

	backoff := 1 * time.Second
	for retries := 0; retries < step.MaxRetries || step.MaxRetries < 0; retries++ {
		err = step.Func(ctx, w, stage, step)
		if err == nil {
			return nil
		}

		if !errors.Is(err, ErrNoConsole) {
			w.Errorf("step [%s] failed with error: %s", step.Name, err)
		}

		if retries+1 < step.MaxRetries {
			w.Errorf("retrying in %s", backoff)
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	return err
}
