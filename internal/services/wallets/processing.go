package wallets

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_processing"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProcessingWallets struct {
	config    *config.Config
	store     store.IStore
	validator *validator.Validate
	sdk       *walletsdk.SDK
}

func newProcessingWallets(
	conf *config.Config,
	store store.IStore,
	validator *validator.Validate,
	sdk *walletsdk.SDK,
) *ProcessingWallets {
	return &ProcessingWallets{
		config:    conf,
		store:     store,
		validator: validator,
		sdk:       sdk,
	}
}

type CreateProcessingWalletParams struct {
	OwnerID    uuid.UUID
	Blockchain wconstants.BlockchainType
	Mnemonic   string
	Passphrase string
}

// Validate validates the CreateProcessingWalletParams fields.
func (p CreateProcessingWalletParams) Validate() error {
	if p.OwnerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if !p.Blockchain.Valid() {
		return fmt.Errorf("invalid blockchain: %s", p.Blockchain.String())
	}

	if p.Mnemonic == "" {
		return fmt.Errorf("mnemonic is empty")
	}

	// if p.Passphrase == "" {
	// 	return fmt.Errorf("passphrase is empty")
	// }

	return nil
}

// Create creates a processing wallet
func (s *ProcessingWallets) Create(ctx context.Context, params CreateProcessingWalletParams, opts ...repos.Option) (*models.ProcessingWallet, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	addressType, err := AddressTypeByBlockchain(params.Blockchain)
	if err != nil {
		return nil, fmt.Errorf("get address type: %w", err)
	}

	// get the max sequence of the processing wallets
	sequence, err := s.store.Wallets().Common().MaxSequence(ctx, params.Blockchain, params.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("get sequence of %s: %w", params.Blockchain.String(), err)
	}

	nextSequence := sequence + 1

	mnemonic := params.Mnemonic
	if s.config.IsEnabledSeedEncryption() {
		// decompress mnemonic
		mnemonic, err = encryption.Decrypt(mnemonic, params.OwnerID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	// generate wallet address
	address, err := s.sdk.AddressWallet(params.Blockchain, addressType, mnemonic, params.Passphrase, uint32(nextSequence)) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("generate adresses: %w", err)
	}

	createParams := repo_wallets_processing.CreateParams{
		Blockchain: params.Blockchain,
		Address:    address,
		OwnerID:    params.OwnerID,
		Sequence:   nextSequence,
		IsActive:   true,
	}

	// check if address is already taken by another owner
	isTaken, err := s.store.Wallets().Processing().IsTakenByAnotherOwner(ctx, params.OwnerID, address)
	if err != nil {
		return nil, fmt.Errorf("check if exists: %w", err)
	}

	// there is a very huge kostyl' here, I resisted but the forces from outside turned out to be stronger
	if isTaken {
		return nil, fmt.Errorf("internal error, try again later")
	}

	// validate create params
	if err := s.validator.Struct(createParams); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	newItem, err := s.store.Wallets().Processing(opts...).Create(ctx, createParams)
	if err != nil {
		return nil, err
	}

	s.store.Cache().ProcessingWallets().Store(cacherKey(params.Blockchain, address), newItem)

	return newItem, nil
}

type FindProcessingWalletsParams = repo_wallets_processing.FindParams

// Find returns processing wallets filtered by params.
func (s *ProcessingWallets) Find(ctx context.Context, params FindProcessingWalletsParams) (*storecmn.FindResponse[*models.ProcessingWallet], error) {
	return s.store.Wallets().Processing().Find(ctx, params)
}

// Get returns the processing wallet by blockchain and address.
func (s *ProcessingWallets) Get(ctx context.Context, blockchain wconstants.BlockchainType, address string) (*models.ProcessingWallet, error) {
	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain.String())
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	return s.store.Wallets().Processing().Get(ctx, blockchain, address)
}

// GetByOwnerID returns the processing wallet by ownerID, blockchain and address.
func (s *ProcessingWallets) GetByOwnerID(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType, address string) (*models.ProcessingWallet, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain.String())
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	return s.store.Wallets().Processing().GetByOwnerID(ctx, ownerID, blockchain, address)
}

// GetAllByOwnerID returns all processing wallets for the owner.
func (s *ProcessingWallets) GetAllByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*models.ProcessingWallet, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}
	return s.store.Wallets().Processing().GetAllByOwnerID(ctx, ownerID)
}

// GetByBlockchain
func (s *ProcessingWallets) GetByBlockchain(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType) (*models.ProcessingWallet, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain.String())
	}

	data, err := s.store.Wallets().Processing().GetByBlockchain(ctx, ownerID, blockchain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

// GetAllPrivateKeys returns all private keys for the owner.
func (s *ProcessingWallets) GetAllPrivateKeys(ctx context.Context, ownerID uuid.UUID, mnemonic, passPhrase string) (*GetAllPrivateKeysResponse, error) {
	wallets, err := s.store.Wallets().Processing().GetAllByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get wallets: %w", err)
	}

	response := make(GetAllPrivateKeysResponse)
	for _, wallet := range wallets {
		if response[wallet.Blockchain] == nil {
			response[wallet.Blockchain] = make([]PrivateKeysItem, 0)
		}

		public, err := s.sdk.AddressPublic(wallet.Blockchain, wallet.Address, mnemonic, passPhrase, uint32(wallet.Sequence)) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("get public: %w", err)
		}

		secret, err := s.sdk.AddressSecret(wallet.Blockchain, wallet.Address, mnemonic, passPhrase, uint32(wallet.Sequence)) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("secret generate: %w", err)
		}

		response[wallet.Blockchain] = append(response[wallet.Blockchain], PrivateKeysItem{
			PublicKey:  public,
			PrivateKey: secret,
			Address:    wallet.Address,
			Kind:       constants.WalletTypeProcessing,
		})
	}

	return &response, nil
}

// GetByBlockchainAndAddress returns a processing wallet by blockchain and address.
func (s *ProcessingWallets) GetByBlockchainAndAddress(ctx context.Context, blockchain wconstants.BlockchainType, address string) (*models.ProcessingWallet, error) {
	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain)
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	data, err := s.store.Wallets().Processing().GetByBlockchainAndAddress(ctx, blockchain, address)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

// GetAllNotCreatedWallets returns all not created wallets.
func (s *ProcessingWallets) GetAllNotCreatedWallets(ctx context.Context) ([]*repo_wallets_processing.GetAllNotCreatedWalletsRow, error) {
	return s.store.Wallets().Processing().GetAllNotCreatedWallets(ctx, wconstants.AllBlockchains.Strings())
}

func (s *ProcessingWallets) checkNotCreatedWallets(ctx context.Context) (int, error) {
	notCreatedWallets, err := s.GetAllNotCreatedWallets(ctx)
	if err != nil {
		return 0, fmt.Errorf("get all not created wallets: %w", err)
	}

	for _, wallet := range notCreatedWallets {
		blockchain := wconstants.BlockchainType(wallet.Blockchain)
		if !blockchain.Valid() {
			return 0, fmt.Errorf("invalid blockchain: %s", wallet.Blockchain)
		}

		owner, err := s.store.Owners().GetByID(ctx, wallet.OwnerID)
		if err != nil {
			return 0, fmt.Errorf("get owner: %w", err)
		}

		if _, err := s.Create(ctx, CreateProcessingWalletParams{
			OwnerID:    wallet.OwnerID,
			Blockchain: blockchain,
			Mnemonic:   owner.Mnemonic,
			Passphrase: owner.PassPhrase.String,
		}); err != nil {
			return 0, fmt.Errorf("create wallet for owner %s and blockchain %s: %w", wallet.OwnerID, wallet.Blockchain, err)
		}
	}

	return len(notCreatedWallets), nil
}
