package system

import (
	"context"

	"github.com/dv-net/dv-processing/internal/store"
)

//go:generate mockgen -source=service.go -destination=../../../mocks/mock_service.go
type IService interface {
	CheckDBVersion(ctx context.Context) error
	SystemVersion(_ context.Context) string
	SystemCommit(_ context.Context) string
	ProcessingID(context.Context) (string, error)
	SetDvSecretKey(context.Context, string) error
}

type service struct {
	store   store.IStore
	version string
	commit  string
}

func New(store store.IStore, systemVersion, systemCommit string) IService {
	return &service{store: store, version: systemVersion, commit: systemCommit}
}
