package workflow_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/mx/logger"
	"github.com/stretchr/testify/require"
)

func stageBeforeStart(_ context.Context, stage *workflow.Stage) error {
	fmt.Printf("Starting stage: %s\n", stage.Name)
	return nil
}

func stageAfterComplete(_ context.Context, stage *workflow.Stage) error {
	fmt.Printf("Stage %s completed\n", stage.Name)
	return nil
}

func stepBeforeStart(_ context.Context, step *workflow.Step) error {
	fmt.Printf("Starting step: %s\n", step.Name)
	return nil
}

func stepAfterComplete(_ context.Context, step *workflow.Step) error {
	fmt.Printf("Step %s completed\n\n", step.Name)
	return nil
}

// Example step functions
func failedStep(_ context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	return fmt.Errorf("failed step")
}

func simpleStep(_ context.Context, _ *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	// fmt.Printf("Executing step: %s\n", step.Name)
	time.Sleep(300 * time.Millisecond)
	return nil
}

func stepWithErr(_ context.Context, _ *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	fmt.Printf("Executing step: %s\n", step.Name)
	if rand.IntN(100) > 1 { //nolint:gosec
		return errors.New("step 1.2 error")
	}
	return nil
}

func stepWithErr2(_ context.Context, _ *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	fmt.Printf("Executing step: %s\n", step.Name)
	if rand.IntN(100) > 50 { //nolint:gosec
		return errors.New("step 2.1 error")
	}
	return nil
}

func TestWorflow(t *testing.T) {
	l := logger.New(
		logger.WithLogLevel(logger.LogLevelDebug),
		logger.WithLogFormat(logger.LoggerFormatConsole),
	)

	wf := workflow.New(
		workflow.WithName("Test Workflow"),
		workflow.WithLogger(l),
		workflow.WithBeforeFn(func(_ context.Context, w *workflow.Workflow) error {
			fmt.Printf("Starting workflow: %s\n\n", w.Name)
			return nil
		}),
		workflow.WithAfterFn(func(_ context.Context, w *workflow.Workflow) error {
			fmt.Printf("\nWorkflow %s completed\n", w.Name)
			return nil
		}),
	)

	wf.Stages = []*workflow.Stage{
		{
			Name: "Stage 1",
			Steps: []*workflow.Step{
				workflow.NewStep("Step 1.1", simpleStep, workflow.WithStepBeforeFn(stepBeforeStart), workflow.WithStepAfterFn(stepAfterComplete)),
				{
					Name:       "Step 1.2",
					Func:       stepWithErr,
					Timeout:    time.Millisecond * 10,
					MaxRetries: -1,
					BeforeFn:   stepBeforeStart,
					AfterFn:    stepAfterComplete,
				},
			},
			BeforeFn: stageBeforeStart,
			AfterFn:  stageAfterComplete,
		},
		{
			Name: "Stage 2",
			Steps: []*workflow.Step{
				{
					Name:     "Step 2.1",
					Func:     stepWithErr2,
					BeforeFn: stepBeforeStart,
					AfterFn:  stepAfterComplete,
				},
				workflow.NewStep("Step 2.2", simpleStep, workflow.WithStepBeforeFn(stepBeforeStart), workflow.WithStepAfterFn(stepAfterComplete)),
			},
			BeforeFn: stageBeforeStart,
			AfterFn:  stageAfterComplete,
		},
		workflow.NewStage("Stage 3", workflow.WithStageBeforeFn(stageBeforeStart), workflow.WithStageAfterFn(stageAfterComplete)),
	}

	err := wf.Run(context.Background())
	require.NoError(t, err)
}

var simpleWorkflowSnapshot = `{
  "steps_states": [
    {
      "current_stage": "Stage 1",
      "current_step": "Step 1.1",
      "start_time": "2024-10-02T13:40:39.960708+03:00",
      "end_time": "2024-10-02T13:40:40.261771+03:00",
      "status": "completed"
    },
    {
      "previous_stage": "Stage 1",
      "previous_step": "Step 1.1",
      "current_stage": "Stage 1",
      "current_step": "Step 1.2",
      "start_time": "2024-10-02T13:40:40.26182+03:00",
      "end_time": "2024-10-02T13:40:40.562862+03:00",
      "status": "completed"
    },
    {
      "previous_stage": "Stage 1",
      "previous_step": "Step 1.2",
      "current_stage": "Stage 2",
      "current_step": "Step 2.1",
      "start_time": "2024-10-02T13:40:40.56294+03:00",
      "end_time": "2024-10-02T13:40:41.063041+03:00",
      "status": "completed"
    },
    {
      "current_stage": "Stage 2",
      "current_step": "Skipped step 2.2",
      "start_time": null,
      "end_time": null,
      "status": "skipped"
    },
    {
      "current_stage": "Stage 2",
      "current_step": "Skipped Step 2.3",
      "start_time": null,
      "end_time": null,
      "status": "skipped"
    },
    {
      "current_stage": "Stage 3",
      "current_step": "Step 3.1",
      "start_time": null,
      "end_time": null,
      "status": "skipped"
    },
    {
      "previous_stage": "Stage 2",
      "previous_step": "Step 2.1",
      "current_stage": "Stage 3",
      "current_step": "Step 3.2",
      "start_time": "2024-10-02T13:40:41.063296+03:00",
      "end_time": "2024-10-02T13:40:41.063297+03:00",
      "status": "completed"
    },
    {
      "previous_stage": "Stage 3",
      "previous_step": "Step 3.2",
      "current_stage": "Stage 3",
      "current_step": "Step 3.3",
      "start_time": "2024-10-02T13:40:41.063301+03:00",
      "end_time": "2024-10-02T13:40:41.063302+03:00",
      "status": "completed"
    }
  ],
  "workflow_state": {
    "is_suspended": false,
    "is_completed": false,
    "is_failed": false
  }
}`

func simpleWorkflow() *workflow.Workflow {
	l := logger.New(
		logger.WithLogLevel(logger.LogLevelDebug),
		logger.WithLogFormat(logger.LoggerFormatConsole),
	)

	wf := workflow.New(
		workflow.WithLogger(l),
		workflow.WithDebug(true),
	).SetOnFailureFn(func(_ context.Context, w *workflow.Workflow, err error) error {
		w.Errorf("SetOnFailureFn: workflow failed with error: %s", err)
		return nil
	})

	wf.Stages = []*workflow.Stage{
		{
			Name: "Stage 1",
			Steps: []*workflow.Step{
				workflow.NewStep("Step 1.1", simpleStep),
				workflow.NewStep("Step 1.2", simpleStep),
			},
		},
		{
			Name: "Stage 2",
			Steps: []*workflow.Step{
				workflow.NewStep("Step 2.1", func(_ context.Context, w *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
					time.Sleep(500 * time.Millisecond)
					w.State.SetNextStage("Stage 3")
					w.State.SetNextStep("Step 3.2")
					return nil
				}),
				workflow.NewStep("Skipped step 2.2", failedStep),
				workflow.NewStep("Skipped Step 2.3", failedStep),
			},
		},
		{
			Name: "Stage 3",
			Steps: []*workflow.Step{
				workflow.NewStep("Step 3.1", simpleStep),
				workflow.NewStep("Step 3.2", func(_ context.Context, w *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
					return nil
				}),
				workflow.NewStep("Step 3.3", func(_ context.Context, w *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
					// return fmt.Errorf("Step 3.3 error")
					return nil
				}).SetMaxRetries(5).SetTimeout(500 * time.Millisecond),
				workflow.NewStep("Step 3.4", simpleStep),
			},
		},
	}

	return wf
}

func TestWorkflowSimple(t *testing.T) {
	wf := simpleWorkflow()

	err := wf.Run(context.Background())
	require.NoError(t, err)

	fmt.Println(wf.GetJSONSnapshot())
}

func TestWorkflowSimpleFromSnapshot(t *testing.T) {
	wf := simpleWorkflow()
	err := wf.SetJSONSnapshot(simpleWorkflowSnapshot)
	require.NoError(t, err)

	err = wf.Run(context.Background())
	require.NoError(t, err)

	fmt.Println(wf.GetJSONSnapshot())
}
