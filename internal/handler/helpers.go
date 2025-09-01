package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/interceptors"
	"github.com/google/uuid"

	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"
)

func WithCORS(connectHandler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         7200, // 2 hours in seconds
	})
	return c.Handler(connectHandler)
}

func WithClientContext[T any](ctx context.Context, request connect.Request[T]) context.Context {
	clientID, err := uuid.Parse(request.Header().Get(interceptors.ClientIDHeaderName))
	if err != nil {
		return ctx
	}

	return constants.WithClientContext(ctx, clientID)
}
