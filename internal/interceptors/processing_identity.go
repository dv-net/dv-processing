package interceptors

import (
	"context"

	"github.com/dv-net/dv-processing/internal/constants"

	"connectrpc.com/connect"
)

type ProcessingIdentityInterceptor struct {
	ID, version string
}

func NewProcessingIdentity(
	processingID, version string,
) *ProcessingIdentityInterceptor {
	return &ProcessingIdentityInterceptor{
		ID: processingID, version: version,
	}
}

// WrapUnary wraps the unary function
func (i *ProcessingIdentityInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		if req.Spec().IsClient {
			req.Header().Set(constants.ProcessingIDParamName.String(), i.ID)
			req.Header().Set(constants.ProcessingVersionParamName.String(), i.version)

			if clientID := i.getClientIDFromContext(ctx); clientID != "" {
				req.Header().Set(constants.ProcessingClientIDParamName.String(), clientID)
			}
		}

		return next(ctx, req)
	}
}

// WrapStreamingClient wraps the streaming server function
func (i *ProcessingIdentityInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(constants.ProcessingIDParamName.String(), i.ID)
		conn.RequestHeader().Set(constants.ProcessingVersionParamName.String(), i.version)

		if clientID := i.getClientIDFromContext(ctx); clientID != "" {
			conn.RequestHeader().Set(constants.ProcessingClientIDParamName.String(), clientID)
		}

		return conn
	}
}

// WrapStreamingHandler wraps the streaming server function
func (i *ProcessingIdentityInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		strHandler connect.StreamingHandlerConn,
	) error {
		return next(ctx, strHandler)
	}
}

func (i *ProcessingIdentityInterceptor) getClientIDFromContext(ctx context.Context) string {
	var clientID string
	clCtx, ok := ctx.Value(constants.ClientContextKey).(constants.ClientCtx)
	if ok {
		clientID = clCtx.ClientID.String()
	}

	return clientID
}
