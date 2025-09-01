package wallets

import (
	"context"
	"sync/atomic"

	"github.com/dv-net/dv-processing/internal/dispatcher"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/valid"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/mx/logger"
)

type Service struct {
	logger logger.Logger
	config *config.Config

	store store.IStore

	coldWallet        *ColdWallets
	hotWallets        *HotWallets
	processingWallets *ProcessingWallets

	cacherInUse      atomic.Bool
	cacheLastLogTime atomic.Int64

	sdk *walletsdk.SDK
}

func New(
	logger logger.Logger,
	conf *config.Config,
	st store.IStore,
	publisher dispatcher.IService,
	sdk *walletsdk.SDK,
) *Service {
	vl := valid.New()

	return &Service{
		logger:            logger,
		config:            conf,
		store:             st,
		sdk:               sdk,
		coldWallet:        newColdWallets(st, vl, sdk),
		hotWallets:        newHotWallets(conf, st, vl, sdk, publisher),
		processingWallets: newProcessingWallets(conf, st, vl, sdk),
	}
}

// SDK
func (s *Service) SDK() *walletsdk.SDK { return s.sdk }

func (s *Service) Cold() *ColdWallets             { return s.coldWallet }
func (s *Service) Hot() *HotWallets               { return s.hotWallets }
func (s *Service) Processing() *ProcessingWallets { return s.processingWallets }

// Name
func (s *Service) Name() string { return "wallets" }

// Start
func (s *Service) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.updateCacheWrapper(ctx, s.config.UseCacheForWallets)
	}()

	go func() {
		s.logger.Info("checking owners not created processing wallets")
		count, err := s.processingWallets.checkNotCreatedWallets(ctx)
		if err != nil {
			s.logger.Errorf("failed to check not created wallets: %s", err)
			return
		}
		s.logger.Infof("checking not created wallets done, new wallets count: %d", count)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// Stop
func (s *Service) Stop(_ context.Context) error { return nil }
