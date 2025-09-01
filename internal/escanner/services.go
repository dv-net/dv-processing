package escanner

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/mx/logger"
)

// EScanner represents a scanner for explorers throw explorer proxy service.
type EScanner struct {
	logger logger.Logger
	store  store.IStore
	config config.Blockchain
	bs     baseservices.IBaseServices
	tm     *taskmanager.TaskManager

	sdk *walletsdk.SDK

	explorers map[wconstants.BlockchainType]struct{}
}

// New creates a new instance of EScanner.
func New(
	logger logger.Logger,
	conf config.Blockchain,
	st store.IStore,
	bs baseservices.IBaseServices,
	tm *taskmanager.TaskManager,
	sdk *walletsdk.SDK,
) *EScanner {
	return &EScanner{
		logger:    logger,
		store:     st,
		config:    conf,
		bs:        bs,
		tm:        tm,
		sdk:       sdk,
		explorers: make(map[wconstants.BlockchainType]struct{}),
	}
}

// Name
func (s *EScanner) Name() string { return "explorer-scanner" }

// Start
func (s *EScanner) Start(ctx context.Context) error {
	// init explorers
	if err := s.initExplorers(ctx); err != nil {
		return fmt.Errorf("init explorers: %w", err)
	}

	return nil
}

// Stop
func (s *EScanner) Stop(_ context.Context) error { return nil }
