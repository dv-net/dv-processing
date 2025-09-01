package wallets

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_cold"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ColdWallets struct {
	store     store.IStore
	validator *validator.Validate
	sdk       *walletsdk.SDK
}

func newColdWallets(
	store store.IStore,
	validator *validator.Validate,
	sdk *walletsdk.SDK,
) *ColdWallets {
	return &ColdWallets{
		validator: validator,
		store:     store,
		sdk:       sdk,
	}
}

type CreateColdWalletParams struct {
	Blockchain wconstants.BlockchainType
	Address    string
	OwnerID    uuid.UUID
}

// Create creates a cold wallet
func (s *ColdWallets) Create(ctx context.Context, params CreateColdWalletParams, opts ...repos.Option) (*models.ColdWallet, error) {
	createParams := repo_wallets_cold.CreateParams{
		Blockchain: params.Blockchain,
		OwnerID:    params.OwnerID,
		Address:    params.Address,
		IsActive:   true,
	}

	// validate create params
	if err := s.validator.Struct(createParams); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if params.Blockchain == wconstants.BlockchainTypeBitcoinCash {
		ok, err := bch.IsLegacyAddress(params.Address, s.sdk.BCH.ChainParams())
		if err != nil {
			return nil, err
		}

		if ok {
			return nil, fmt.Errorf("legacy address is not allowed")
		}
	}

	newItem, err := s.store.Wallets().Cold(opts...).Create(ctx, createParams)
	if err != nil {
		return nil, err
	}

	s.store.Cache().ColdWallets().Store(cacherKey(params.Blockchain, params.Address), newItem)

	return newItem, nil
}

// BatchAttachColdWallets creates cold wallets in batch.
func (s *ColdWallets) BatchAttachColdWallets(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType, params []CreateColdWalletParams) error {
	if ownerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return fmt.Errorf("invalid blockchain: %s", blockchain)
	}

	err := pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := s.DeleteAllByBlockchainAndOwnerID(ctx, DeleteAllColdWalletsParams{
			OwnerID:    &ownerID,
			Blockchain: &blockchain,
		}, repos.WithTx(tx)); err != nil {
			return fmt.Errorf("delete cold wallets: %w", err)
		}

		for _, param := range params {
			_, err := s.Create(ctx, param, repos.WithTx(tx))
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// GetAllByOwnerID returns all hot wallets for the owner.
func (s *ColdWallets) GetAllByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*models.ColdWallet, error) {
	return s.store.Wallets().Cold().GetAllByOwnerID(ctx, ownerID)
}

// Get returns a cold wallet by ownerID, address and blockchain.
func (s *ColdWallets) Get(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType, address string) (*models.ColdWallet, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain)
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	data, err := s.store.Wallets().Cold().Get(ctx, repo_wallets_cold.GetParams{
		OwnerID:    ownerID,
		Blockchain: blockchain,
		Address:    address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

type FindColdWalletsParams = repo_wallets_cold.FindParams

// Find returns cold wallets filtered by params.
func (s *ColdWallets) Find(ctx context.Context, params FindColdWalletsParams) (*storecmn.FindResponse[*models.ColdWallet], error) {
	return s.store.Wallets().Cold().Find(ctx, params)
}

type DeleteAllColdWalletsParams = repo_wallets_cold.DeleteAllByOwnerIDParams

// DeleteAllByBlockchainAndOwnerID deletes cold wallets by params.
func (s *ColdWallets) DeleteAllByBlockchainAndOwnerID(ctx context.Context, params DeleteAllColdWalletsParams, opts ...repos.Option) error {
	// TODO: remove pointwise
	s.store.Cache().ColdWallets().Clear()
	return s.store.Wallets().Cold(opts...).Delete(ctx, params)
}

// GetByBlockchainAndAddress returns a cold wallet by blockchain and address.
func (s *ColdWallets) GetByBlockchainAndAddress(ctx context.Context, blockchain wconstants.BlockchainType, address string) (*models.ColdWallet, error) {
	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain)
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	data, err := s.store.Wallets().Cold().GetByBlockchainAndAddress(ctx, blockchain, address)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}
