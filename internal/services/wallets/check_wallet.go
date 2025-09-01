package wallets

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/google/uuid"
)

var ErrAddressNotFound = fmt.Errorf("address not found")

type CheckWalletResult struct {
	WalletType       constants.WalletType
	OwnerID          uuid.UUID
	ExternalWalletID *string
	IsActivated      *bool
}

func (s *CheckWalletResult) Activated() bool {
	return s.IsActivated != nil && *s.IsActivated
}

// CheckWallet - determines whether the address belongs to us or not.
// checks hot, cold, and processing wallets.
// returns the wallet type and the owner id
func (s *Service) CheckWallet(ctx context.Context, blockchain wconstants.BlockchainType, address string) (*CheckWalletResult, error) {
	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	if s.config.UseCacheForWallets {
		return s.checkWalletInCache(blockchain, address)
	}

	return s.checkWalletInDatabase(ctx, blockchain, address)
}

// checkWalletInCache
func (s *Service) checkWalletInCache(blockchain wconstants.BlockchainType, address string) (*CheckWalletResult, error) {
	// check if the address is a hot wallet
	{
		wallet, ok := s.store.Cache().HotWallets().Load(cacherKey(blockchain, address))
		if ok {
			return &CheckWalletResult{
				WalletType:       constants.WalletTypeHot,
				OwnerID:          wallet.OwnerID,
				ExternalWalletID: &wallet.ExternalWalletID,
				IsActivated:      &wallet.IsActivated,
			}, nil
		}
	}

	// check if the address is a processing wallet
	{
		wallet, ok := s.store.Cache().ProcessingWallets().Load(cacherKey(blockchain, address))
		if ok {
			return &CheckWalletResult{
				WalletType: constants.WalletTypeProcessing,
				OwnerID:    wallet.OwnerID,
			}, nil
		}
	}

	// check if the address is a cold wallet
	{
		wallet, ok := s.store.Cache().ColdWallets().Load(cacherKey(blockchain, address))
		if ok {
			return &CheckWalletResult{
				WalletType: constants.WalletTypeCold,
				OwnerID:    wallet.OwnerID,
			}, nil
		}
	}

	return nil, fmt.Errorf("check wallet in cache: %w", ErrAddressNotFound)
}

// checkWalletInDatabase
func (s *Service) checkWalletInDatabase(ctx context.Context, blockchain wconstants.BlockchainType, address string) (*CheckWalletResult, error) {
	// check if the address is a hot wallet
	{
		res, err := s.Hot().GetByBlockchainAndAddress(ctx, blockchain, address)
		if err != nil && !errors.Is(err, storecmn.ErrNotFound) {
			return nil, fmt.Errorf("find hot wallets: %w", err)
		}

		if err == nil {
			return &CheckWalletResult{
				WalletType:       constants.WalletTypeHot,
				OwnerID:          res.OwnerID,
				ExternalWalletID: &res.ExternalWalletID,
				IsActivated:      &res.IsActivated,
			}, nil
		}
	}

	// check if the address is a processing wallet
	{
		res, err := s.Processing().GetByBlockchainAndAddress(ctx, blockchain, address)
		if err != nil && !errors.Is(err, storecmn.ErrNotFound) {
			return nil, fmt.Errorf("find processing wallets: %w", err)
		}

		if err == nil {
			return &CheckWalletResult{
				WalletType: constants.WalletTypeProcessing,
				OwnerID:    res.OwnerID,
			}, nil
		}
	}

	// check if the address is a cold wallet
	{
		res, err := s.Cold().GetByBlockchainAndAddress(ctx, blockchain, address)
		if err != nil && !errors.Is(err, storecmn.ErrNotFound) {
			return nil, fmt.Errorf("find cold wallets: %w", err)
		}

		if err == nil {
			return &CheckWalletResult{
				WalletType: constants.WalletTypeCold,
				OwnerID:    res.OwnerID,
			}, nil
		}
	}

	return nil, fmt.Errorf("check wallet in database: %w", ErrAddressNotFound)
}
