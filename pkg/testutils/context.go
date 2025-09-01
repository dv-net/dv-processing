package testutils

import (
	"context"

	"github.com/dv-net/dv-processing/internal/constants"
)

func GetContext() context.Context {
	appCtx := context.WithValue(context.Background(), constants.ProcessingIDParamName, "55116b30-4700-465b-96fa-b657bbb7a5d0")
	return context.WithValue(appCtx, constants.ProcessingVersionParamName, "local-test")
}
