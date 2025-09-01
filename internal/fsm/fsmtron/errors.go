package fsmtron

import (
	"encoding/json"
)

type FailedTransferError struct {
	FailedStageName string `json:"failed_stage_name,omitempty"`
	FailedStepName  string `json:"failed_step_name,omitempty"`
	Msg             string `json:"msg,omitempty"`
	err             error
}

func (e *FailedTransferError) FailedStage() string {
	return e.FailedStageName
}

func (e *FailedTransferError) FailedStep() string {
	return e.FailedStepName
}

func (e *FailedTransferError) Error() string {
	return e.err.Error()
}

func (e *FailedTransferError) MarshallJSON() ([]byte, error) {
	return json.Marshal(e)
}

func newErrorFailedTransfer(err error, stepName, stageName string) error {
	return &FailedTransferError{
		FailedStageName: stageName,
		FailedStepName:  stepName,
		Msg:             err.Error(),
		err:             err,
	}
}
