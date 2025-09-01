package constants

import (
	"context"
	"errors"
	"fmt"
)

type ProcessingContextKeyType string

func (p ProcessingContextKeyType) String() string {
	return string(p)
}

const (
	ProcessingIDParamName       ProcessingContextKeyType = "X-Processing-ID"
	ProcessingVersionParamName  ProcessingContextKeyType = "X-Processing-Version"
	ProcessingClientIDParamName ProcessingContextKeyType = "X-Processing-Client-ID"
)

type ProcessingIdentity struct {
	ID, Version string
}

func IdentityFromContext(ctx context.Context) (ProcessingIdentity, error) {
	id, ok := ctx.Value(ProcessingIDParamName).(string)
	if !ok {
		return ProcessingIdentity{}, errors.New("undefined processing ID")
	}

	version, ok := ctx.Value(ProcessingVersionParamName).(string)
	if !ok {
		return ProcessingIdentity{}, fmt.Errorf("undefined processing version")
	}

	return ProcessingIdentity{ID: id, Version: version}, nil
}
