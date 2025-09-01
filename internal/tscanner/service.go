package tscanner

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/mx/logger"
)

// TScanner is a service that scans the blockchain for new transfers.
type TScanner struct {
	logger logger.Logger

	store store.IStore
	bs    baseservices.IBaseServices
	tm    *taskmanager.TaskManager

	inUse atomic.Bool
}

func New(
	l logger.Logger,
	store store.IStore,
	bs baseservices.IBaseServices,
	tm *taskmanager.TaskManager,
) *TScanner {
	return &TScanner{
		logger: logger.With(l, "service", "transfers-scanner"),
		store:  store,
		bs:     bs,
		tm:     tm,
	}
}

// Name
func (s *TScanner) Name() string { return "transfers-scanner" }

// Start
func (s *TScanner) Start(ctx context.Context) error {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			go func() {
				if err := s.processAllNewTransfers(ctx); err != nil {
					s.logger.Error(err)
				}
			}()
		}
	}
}

// Stop
func (s *TScanner) Stop(_ context.Context) error {
	return nil
}
