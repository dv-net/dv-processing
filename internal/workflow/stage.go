package workflow

import "context"

// Stage represents a unique stage that can contain multiple steps.
type Stage struct {
	// Name is the name of the stage.
	Name string
	// Steps is the list of steps for the stage.
	Steps []*Step
	// BeforeFn is the before start function for the stage.
	BeforeFn func(context.Context, *Stage) error
	// AfterFn is the after complete function for the stage.
	AfterFn func(context.Context, *Stage) error
}

// NewStage returns a new stage.
func NewStage(name string, opts ...StageOption) *Stage {
	s := &Stage{
		Name: name,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
