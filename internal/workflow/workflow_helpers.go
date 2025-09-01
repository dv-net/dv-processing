package workflow

import "fmt"

func (w *Workflow) checkStatus() bool {
	res := w.State.IsCompleted || w.State.IsSuspended || w.State.IsFailed

	if w.State.IsCompleted {
		w.Debugf("workflow is completed, skipping")
	}

	if w.State.IsSuspended {
		w.Debugf("workflow is suspended, skipping")
	}

	if w.State.IsFailed {
		w.Debugf("workflow is failed, skipping")
	}

	return res
}

// commonLogFields returns the common log fields for the workflow.
func (w *Workflow) commonLogFields() []any {
	fields := []any{}

	if w.Name != "" {
		fields = append(fields, "workflow", w.Name)
	}

	if w.State.NextStage != "" {
		fields = append(fields, "next_stage", w.State.NextStage)
	}

	if w.State.NextStep != "" {
		fields = append(fields, "next_step", w.State.NextStep)
	}

	if w.State.IsSuspended {
		fields = append(fields, "is_suspended", w.State.IsSuspended)
	}

	if w.State.IsCompleted {
		fields = append(fields, "is_completed", w.State.IsCompleted)
	}

	if w.State.IsFailed {
		fields = append(fields, "is_failed", w.State.IsFailed)
	}

	return fields
}

// Debugf logs the debug message.
func (w *Workflow) Debugf(format string, args ...any) {
	if w.Debug && w.logger != nil {
		w.logger.Debugw(fmt.Sprintf(format, args...), w.commonLogFields()...)
	}
}

// infof logs the info message.
func (w *Workflow) Infof(format string, args ...any) {
	if w.logger != nil {
		w.logger.Infow(fmt.Sprintf(format, args...), w.commonLogFields()...)
	}
}

// Warnf logs the warning message.
func (w *Workflow) Warnf(format string, args ...any) {
	if w.logger != nil {
		w.logger.Warnw(fmt.Sprintf(format, args...), w.commonLogFields()...)
	}
}

// Errorf logs the error message.
func (w *Workflow) Errorf(format string, args ...any) {
	if w.logger != nil {
		w.logger.Errorw(fmt.Sprintf(format, args...), w.commonLogFields()...)
	}
}
