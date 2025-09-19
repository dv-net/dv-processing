package system

import (
	"context"

	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/mx/logger"
)

//go:generate mockgen -source=service.go -destination=../../../mocks/mock_service.go
type IService interface {
	CheckDBVersion(ctx context.Context) error
	SystemVersion(_ context.Context) string
	SystemCommit(_ context.Context) string
	ProcessingID(context.Context) (string, error)
	SetDvSecretKey(context.Context, string) error
	GetLogs(_ context.Context) ([]logger.MemoryLog, error)
}

type service struct {
	logger  logger.ExtendedLogger
	store   store.IStore
	version string
	commit  string
}

func New(l logger.ExtendedLogger, store store.IStore, systemVersion, systemCommit string) IService {
	return &service{logger: l, store: store, version: systemVersion, commit: systemCommit}
}
