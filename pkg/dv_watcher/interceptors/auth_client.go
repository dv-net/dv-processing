package interceptors

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const ClientContextKey = "client_ctx"

const (
	headerNameClientID = "X-Client-ID"
	headerNameSecret   = "X-Client-Secret"
)

type ConnectType string

type Client struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	ConnectType ConnectType      `db:"connect_type" json:"connect_type"`
	CallbackURL pgtype.Text      `db:"callback_url" json:"callback_url"`
	CreatedAt   pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt   pgtype.Timestamp `db:"updated_at" json:"updated_at"`
}

const (
	ConnectTypeGRPC ConnectType = "grpc"
	ConnectTypeRest ConnectType = "rest"
)

func (ct ConnectType) IsValid() bool {
	switch ct {
	case ConnectTypeGRPC, ConnectTypeRest:
		return true
	default:
		return false
	}
}

type WatcherAuthInterceptor struct {
	clientID, clientSecret string
}

type ClientContext struct {
	Client *Client
}

func NewWatcherAuthInterceptor(
	clientID, clientSecret string,
) *WatcherAuthInterceptor {
	return &WatcherAuthInterceptor{
		clientID: clientID, clientSecret: clientSecret,
	}
}

// WrapUnary wraps the unary function
func (i *WatcherAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		if req.Spec().IsClient {
			req.Header().Set(headerNameClientID, i.clientID)
			req.Header().Set(headerNameSecret, i.clientSecret)
		}

		return next(ctx, req)
	}
}

// WrapStreamingClient wraps the streaming server function
func (i *WatcherAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(headerNameClientID, i.clientID)
		conn.RequestHeader().Set(headerNameSecret, i.clientSecret)

		return conn
	}
}

// WrapStreamingHandler wraps the streaming server function
func (i *WatcherAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		strHandler connect.StreamingHandlerConn,
	) error {
		return next(ctx, strHandler)
	}
}
