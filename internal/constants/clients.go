package constants

import (
	"context"

	"github.com/google/uuid"
)

type ClientContextKeyType string

const (
	ClientContextKey ClientContextKeyType = "client_context"
)

type ClientCtx struct {
	ClientID uuid.UUID
}

func WithClientContext(ctx context.Context, clientID uuid.UUID) context.Context {
	return context.WithValue(ctx, ClientContextKey, ClientCtx{clientID})
}

func GetClientIDFromContext(ctx context.Context) string {
	var clientID string
	clCtx, ok := ctx.Value(ClientContextKey).(ClientCtx)
	if ok {
		clientID = clCtx.ClientID.String()
	}

	return clientID
}
