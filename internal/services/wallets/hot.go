package wallets

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/dispatcher"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_hot"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type HotWallets struct {
	config    *config.Config
	publisher dispatcher.IService
	store     store.IStore
	validator *validator.Validate
	sdk       *walletsdk.SDK
}

func newHotWallets(
	conf *config.Config,
	store store.IStore,
	validator *validator.Validate,
	sdk *walletsdk.SDK,
	publisher dispatcher.IService,
) *HotWallets {
	return &HotWallets{
		config:    conf,
		store:     store,
		validator: validator,
		sdk:       sdk,
		publisher: publisher,
	}
}

type CreateHotWalletParams struct {
	OwnerID          uuid.UUID                 `validate:"required,uuid4"`
	Blockchain       wconstants.BlockchainType `validate:"required"`
	AddressType      string                    `validate:"required"`
	Mnemonic         string                    `validate:"required"`
	Passphrase       string
	ExternalWalletID string `validate:"required"`
}

// Create creates a hot wallet
func (s *HotWallets) Create(ctx context.Context, params CreateHotWalletParams, opts ...repos.Option) (*models.HotWallet, error) {
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
	address, err := s.sdk.AddressWallet(params.Blockchain, params.AddressType, mnemonic, params.Passphrase, uint32(nextSequence)) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("generate adresses: %w", err)
	}

	createParams := repo_wallets_hot.CreateParams{
		Blockchain:       params.Blockchain,
		Address:          address,
		OwnerID:          params.OwnerID,
		ExternalWalletID: params.ExternalWalletID,
		Sequence:         nextSequence,
		IsActive:         true,
	}

	// validate create params
	if err := s.validator.Struct(createParams); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// check evm address and use the same address for all evm blockchains
	if params.Blockchain.IsEVM() {
		existsEVMAddresses, err := s.store.Wallets().Hot().FindEVMByExternalID(ctx,
			params.ExternalWalletID,
			wconstants.EVMBlockchains().Strings(),
			params.OwnerID,
		)
		if err != nil {
			return nil, fmt.Errorf("find evm address by external id: %w", err)
		}

		if len(existsEVMAddresses) > 0 {
			existsItem := existsEVMAddresses[0]
			createParams.Address = existsItem.Address
			createParams.Sequence = existsItem.Sequence
		}
	}

	newItem, err := s.store.Wallets().Hot(opts...).Create(ctx, createParams)
	if err != nil {
		return nil, err
	}

	s.store.Cache().HotWallets().Store(cacherKey(params.Blockchain, address), newItem)

	// publish new hot wallet created event
	go func() {
		s.publisher.CreatedHotWalletDispatcher().Publish(newItem)
	}()

	return newItem, nil
}

// Get returns the hot wallet by ownerID, blockchain and address.
func (s *HotWallets) Get(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType, address string) (*models.HotWallet, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain.String())
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	return s.store.Wallets().Hot().Get(ctx, ownerID, blockchain, address)
}

// GetAllByOwnerID returns all hot wallets for the owner.
func (s *HotWallets) GetAllByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*models.HotWallet, error) {
	return s.store.Wallets().Hot().GetAllByOwnerID(ctx, ownerID)
}

// GetAllPrivateKeys returns all private keys for the owner.
func (s *HotWallets) GetAllPrivateKeys(ctx context.Context, ownerID uuid.UUID, mnemonic, passPhrase string) (*GetAllPrivateKeysResponse, error) {
	wallets, err := s.store.Wallets().Hot().GetAllByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get wallets: %w", err)
	}

	if s.config.IsEnabledSeedEncryption() {
		mnemonic, err = encryption.Decrypt(mnemonic, ownerID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
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
			Kind:       constants.WalletTypeHot,
		})
	}

	return &response, nil
}

type GetPrivateKeysByIDParams struct {
	OwnerID    uuid.UUID
	Addresses  []string
	Mnemonic   string
	Passphrase string
}

func (s *HotWallets) GetPrivateKeysByIDs(ctx context.Context, params *GetPrivateKeysByIDParams) (*GetAllPrivateKeysResponse, error) {
	wallets, err := s.store.Wallets().Hot().GetManyByOwnerAndWalletAddresses(ctx, params.Addresses, params.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("get wallets: %w", err)
	}

	mnemonic := params.Mnemonic
	if s.config.IsEnabledSeedEncryption() {
		mnemonic, err = encryption.Decrypt(mnemonic, params.OwnerID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	response := make(GetAllPrivateKeysResponse)
	for _, wallet := range wallets {
		if response[wallet.Blockchain] == nil {
			response[wallet.Blockchain] = make([]PrivateKeysItem, 0)
		}

		public, err := s.sdk.AddressPublic(wallet.Blockchain, wallet.Address, mnemonic, params.Passphrase, uint32(wallet.Sequence)) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("get public: %w", err)
		}

		secret, err := s.sdk.AddressSecret(wallet.Blockchain, wallet.Address, mnemonic, params.Passphrase, uint32(wallet.Sequence)) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("secret generate: %w", err)
		}

		response[wallet.Blockchain] = append(response[wallet.Blockchain], PrivateKeysItem{
			PublicKey:  public,
			PrivateKey: secret,
			Address:    wallet.Address,
			Kind:       constants.WalletTypeHot,
		})
	}
	return &response, nil
}

type FindHotWalletsParams = repo_wallets_hot.FindParams

// Find returns hot wallets filtered by params.
func (s *HotWallets) Find(ctx context.Context, params FindHotWalletsParams) (*storecmn.FindResponse[*models.HotWallet], error) {
	return s.store.Wallets().Hot().Find(ctx, params)
}

// MarkDirty marks the hot wallet as dirty.
func (s *HotWallets) MarkDirty(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType, address string) error {
	if ownerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return fmt.Errorf("invalid blockchain: %s", blockchain.String())
	}

	if address == "" {
		return storecmn.ErrEmptyAddress
	}

	// check if wallet exists
	exists, err := s.store.Wallets().Hot().Exist(ctx, address, blockchain, ownerID)
	if err != nil {
		return fmt.Errorf("check if wallet exists: %w", err)
	}

	if !exists {
		return storecmn.ErrNotFound
	}

	return s.store.Wallets().Hot().MarkDirty(ctx, blockchain, address, ownerID)
}

// ActivateWallet activates the hot wallet.
func (s *HotWallets) ActivateWallet(ctx context.Context, ownerID uuid.UUID, blockchain wconstants.BlockchainType, address string, opts ...repos.Option) error {
	if ownerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if !blockchain.Valid() {
		return fmt.Errorf("invalid blockchain: %s", blockchain.String())
	}

	if address == "" {
		return storecmn.ErrEmptyAddress
	}

	return s.store.Wallets().Hot(opts...).ActivateWallet(ctx, blockchain, address, ownerID)
}

// GetByBlockchainAndAddress returns a hot wallet by blockchain and address.
func (s *HotWallets) GetByBlockchainAndAddress(ctx context.Context, blockchain wconstants.BlockchainType, address string) (*models.HotWallet, error) {
	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain: %s", blockchain)
	}

	if address == "" {
		return nil, storecmn.ErrEmptyAddress
	}

	data, err := s.store.Wallets().Hot().GetByBlockchainAndAddress(ctx, blockchain, address)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

// GetAll returns all hot wallets
func (s *HotWallets) GetAll(ctx context.Context) ([]*models.HotWallet, error) {
	res, err := s.store.Wallets().Hot().GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch all hot wallets: %w", err)
	}

	return res, nil
}
