package interceptors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/dv-net/dv-processing/api/processing/client/v1/clientv1connect"
	"github.com/dv-net/dv-processing/internal/services/clients"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/internal/util"
	"github.com/google/uuid"
)

const (
	ClientIDHeaderName = "X-Client-ID"
	signHeaderName     = "X-Sign"
)

var (
	errEmptySignKey  = fmt.Errorf("empty sign key")
	errEmptyClientID = fmt.Errorf("empty client id")
)

type SignInterceptor struct {
	clientService       *clients.Service
	disableCheckingSign bool
}

func NewSignInterceptor(
	clientService *clients.Service,
	disableCheckingSign bool,
) *SignInterceptor {
	return &SignInterceptor{
		clientService:       clientService,
		disableCheckingSign: disableCheckingSign,
	}
}

// WrapUnary wraps the unary function
func (i *SignInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		// skip checking sign key
		if i.disableCheckingSign {
			return next(ctx, req)
		}

		// check sign key
		if err := i.checkSignKey(ctx, req); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		return next(ctx, req)
	})
}

// WrapStreamingServer wraps the streaming server function
func (*SignInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		return next(ctx, spec)
	})
}

// WrapStreamingServer wraps the streaming server function
func (i *SignInterceptor) WrapStreamingHandler(_ connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(
		_ context.Context,
		_ connect.StreamingHandlerConn,
	) error {
		return fmt.Errorf("streaming is not supported")
	})
}

// checkSignKey checks the sign key from the metadata
func (i *SignInterceptor) checkSignKey(ctx context.Context, req connect.AnyRequest) error {
	// skip checking sign key for create client
	if req.Spec().Procedure == clientv1connect.ClientServiceCreateProcedure {
		return nil
	}

	// get sign key
	signKey := req.Header().Get(signHeaderName)
	if signKey == "" {
		return errEmptySignKey
	}

	// get client id
	clientID := req.Header().Get(ClientIDHeaderName)
	if clientID == "" {
		return errEmptyClientID
	}

	cui, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("parse client id error: %w", err)
	}

	// get client
	client, err := i.clientService.GetByID(ctx, cui)
	if err != nil {
		if errors.Is(err, storecmn.ErrNotFound) {
			return fmt.Errorf("invalid client")
		}
		return fmt.Errorf("get client: %w", err)
	}

	// marshal payload
	payload, err := json.Marshal(req.Any())
	if err != nil {
		return fmt.Errorf("marshal payload error: %w", err)
	}

	// check sign key
	if signKey != util.SHA256Signature(payload, client.SecretKey) {
		return fmt.Errorf("invalid sign key")
	}

	return nil
}
