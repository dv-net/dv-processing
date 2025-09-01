package wallets

import (
	"context"
	"fmt"
	"time"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"golang.org/x/sync/errgroup"
)

const cacherInterval = time.Second

func (s *Service) updateCacheWrapper(ctx context.Context, isEnabled bool) error {
	if !isEnabled {
		return nil
	}

	ticker := time.NewTicker(cacherInterval)
	defer ticker.Stop()

	// immediately process webhooks after startup service
	go func() {
		if err := s.cacherHandler(ctx); err != nil {
			s.logger.Error(err)
		}
	}()

	// process webhooks by ticker
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			go func() {
				if err := s.cacherHandler(ctx); err != nil {
					s.logger.Error(err)
				}
			}()
		}
	}
}

func cacherKey(blockchain wconstants.BlockchainType, address string) string {
	return blockchain.String() + "-" + address
}

func (s *Service) cacherHandler(ctx context.Context) error {
	if !s.cacherInUse.CompareAndSwap(false, true) {
		return nil
	}
	defer s.cacherInUse.Store(false)

	eg, egCtx := errgroup.WithContext(ctx)

	var dbHotWalletsLength int
	var dbProcessingWalletsLength int
	var dbColdWalletsLength int

	// process hot wallets
	eg.Go(func() error { //nolint:dupl
		hotWallets, err := s.store.Wallets().Hot().Find(egCtx, FindHotWalletsParams{})
		if err != nil {
			return fmt.Errorf("find hot wallets: %w", err)
		}

		dbHotWalletsLength = len(hotWallets.Items)

		updatedHotWallets := make(map[string]struct{})
		for _, wallet := range hotWallets.Items {
			key := cacherKey(wallet.Blockchain, wallet.Address)
			s.store.Cache().HotWallets().Store(key, wallet)
			updatedHotWallets[key] = struct{}{}
		}

		s.store.Cache().HotWallets().Range(func(key string, _ *models.HotWallet) bool {
			if _, ok := updatedHotWallets[key]; !ok {
				s.store.Cache().HotWallets().Delete(key)
			}
			return true
		})
		return nil
	})

	// process processing wallets
	eg.Go(func() error { //nolint:dupl
		processingWallets, err := s.store.Wallets().Processing().Find(egCtx, FindProcessingWalletsParams{})
		if err != nil {
			return fmt.Errorf("find processing wallets: %w", err)
		}

		dbProcessingWalletsLength = len(processingWallets.Items)

		updatedProcessingWallets := make(map[string]struct{})
		for _, wallet := range processingWallets.Items {
			key := cacherKey(wallet.Blockchain, wallet.Address)
			s.store.Cache().ProcessingWallets().Store(key, wallet)
			updatedProcessingWallets[key] = struct{}{}
		}

		s.store.Cache().ProcessingWallets().Range(func(key string, _ *models.ProcessingWallet) bool {
			if _, ok := updatedProcessingWallets[key]; !ok {
				s.store.Cache().ProcessingWallets().Delete(key)
			}
			return true
		})

		return nil
	})

	// process cold wallets
	eg.Go(func() error { //nolint:dupl
		coldWallets, err := s.store.Wallets().Cold().Find(egCtx, FindColdWalletsParams{})
		if err != nil {
			return fmt.Errorf("find cold wallets: %w", err)
		}

		dbColdWalletsLength = len(coldWallets.Items)

		updatedColdWallets := make(map[string]struct{})
		for _, wallet := range coldWallets.Items {
			key := cacherKey(wallet.Blockchain, wallet.Address)
			s.store.Cache().ColdWallets().Store(key, wallet)
			updatedColdWallets[key] = struct{}{}
		}

		s.store.Cache().ColdWallets().Range(func(key string, _ *models.ColdWallet) bool {
			if _, ok := updatedColdWallets[key]; !ok {
				s.store.Cache().ColdWallets().Delete(key)
			}
			return true
		})
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	now := time.Now()

	// log cache sizes every 5 seconds
	if now.Unix()-s.cacheLastLogTime.Load() > 5 {
		s.logger.Debugw("cacherHandler",
			"hw_cache", s.store.Cache().HotWallets().Size(),
			"hw_db", dbHotWalletsLength,
			"pw_cache", s.store.Cache().ProcessingWallets().Size(),
			"pw_db", dbProcessingWalletsLength,
			"cw_cache", s.store.Cache().ColdWallets().Size(),
			"cw_db", dbColdWalletsLength,
		)
		s.cacheLastLogTime.Store(now.Unix())
	}

	return nil
}
