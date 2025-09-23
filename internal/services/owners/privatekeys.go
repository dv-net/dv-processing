package owners

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/dv-net/dv-processing/internal/models"
	wallets2 "github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

type GetHotWalletKeysResponse map[wconstants.BlockchainType][]PrivateKeyItem

type GetHotWalletKeysRequest struct {
	OwnerID           uuid.UUID `json:"owner_id"`
	WalletAddresses   []string  `json:"wallet_addresses"`
	ExcludedAddresses []string  `json:"excluded_addresses,omitempty"`
	OTP               string    `json:"otp"`
}

func (o *GetHotWalletKeysRequest) Validate() error {
	if o.OwnerID == uuid.Nil {
		return ErrEmptyOwnerID
	}
	if o.OTP == "" {
		return ErrEmptyOTP
	}
	return nil
}

func (s *Service) GetHotWalletKeys(ctx context.Context, request *GetHotWalletKeysRequest) (*GetHotWalletKeysResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}

	owner, err := s.GetByID(ctx, request.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	if !owner.OtpConfirmed {
		return nil, ErrTwoFactorDisabled
	}

	if ok := s.ValidateTwoFactorToken(ctx, owner.ID, request.OTP); ok != nil {
		return nil, fmt.Errorf("validate otp: %w", ok)
	}

	response := make(GetHotWalletKeysResponse)

	// If no wallet addresses provided, get all wallets for the owner
	if len(request.WalletAddresses) == 0 {
		allWallets, err := s.walletsSvc.Hot().GetAllByOwnerID(ctx, request.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("get all wallets: %w", err)
		}
		request.WalletAddresses = lo.Map(allWallets, func(item *models.HotWallet, _ int) string {
			return item.Address
		})
	}

	// Filter out excluded addresses if provided.
	if len(request.ExcludedAddresses) > 0 {
		request.WalletAddresses = lo.Filter(request.WalletAddresses, func(address string, _ int) bool {
			return !lo.Contains(request.ExcludedAddresses, address)
		})
	}

	walletData, err := s.walletsSvc.Hot().GetPrivateKeysByIDs(ctx, &wallets2.GetPrivateKeysByIDParams{
		OwnerID:    request.OwnerID,
		Addresses:  request.WalletAddresses,
		Mnemonic:   owner.Mnemonic,
		Passphrase: owner.PassPhrase.String,
	})
	if err != nil {
		return nil, err
	}

	for blockchain, keys := range *walletData {
		if response[blockchain] == nil {
			hotData := lo.Map(keys, func(item wallets2.PrivateKeysItem, _ int) PrivateKeyItem {
				return PrivateKeyItem{
					PublicKey:  item.PublicKey,
					PrivateKey: item.PrivateKey,
					Address:    item.Address,
				}
			})
			response[blockchain] = hotData
		}
	}

	return &response, nil
}

type GetAllPrivateKeysResponse map[wconstants.BlockchainType][]PrivateKeysItem

type GetAllPrivateKeysRequest struct {
	OwnerID uuid.UUID `json:"owner_id"`
	OTP     string    `json:"otp"`
}

func (o *GetAllPrivateKeysRequest) Validate() error {
	if o.OwnerID == uuid.Nil {
		return ErrEmptyOwnerID
	}
	if o.OTP == "" {
		return ErrEmptyOTP
	}
	return nil
}

// GetAllPrivateKeys returns all private keys for the owner.
func (s *Service) GetAllPrivateKeys(ctx context.Context, request GetAllPrivateKeysRequest) (*GetAllPrivateKeysResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}

	owner, err := s.GetByID(ctx, request.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	if !owner.OtpConfirmed {
		return nil, ErrTwoFactorDisabled
	}

	// Validate as soon as we have it, since it's time based.
	// Also check for remaining request's fields to reduce store calls.
	if ok := s.ValidateTwoFactorToken(ctx, owner.ID, request.OTP); ok != nil {
		return nil, fmt.Errorf("validate otp: %w", ok)
	}

	response := make(GetAllPrivateKeysResponse)

	mnemonic := owner.Mnemonic
	if s.config.IsEnabledSeedEncryption() {
		mnemonic, err = encryption.Decrypt(mnemonic, owner.ID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	// handle private keys
	allProcessingPrivateKeys, err := s.walletsSvc.Processing().GetAllPrivateKeys(ctx, request.OwnerID, mnemonic, owner.PassPhrase.String)
	if err != nil {
		return nil, fmt.Errorf("get processing private keys: %w", err)
	}

	for blockchain, processingPrivateKeys := range *allProcessingPrivateKeys {
		if response[blockchain] == nil {
			response[blockchain] = make([]PrivateKeysItem, 0)
		}

		privateKeys := make([]PrivateKeysItem, len(processingPrivateKeys))
		for i, processingPrivateKey := range processingPrivateKeys {
			privateKeys[i] = PrivateKeysItem(processingPrivateKey)
		}

		response[blockchain] = append(response[blockchain], privateKeys...)
	}

	// handle hot private keys
	allHotPrivateKeys, err := s.walletsSvc.Hot().GetAllPrivateKeys(ctx, request.OwnerID, mnemonic, owner.PassPhrase.String)
	if err != nil {
		return nil, fmt.Errorf("get hot private keys: %w", err)
	}

	for blockchain, hotPrivateKeys := range *allHotPrivateKeys {
		if response[blockchain] == nil {
			response[blockchain] = make([]PrivateKeysItem, 0)
		}

		privateKeys := make([]PrivateKeysItem, len(hotPrivateKeys))
		for i, hotPrivateKey := range hotPrivateKeys {
			privateKeys[i] = PrivateKeysItem(hotPrivateKey)
		}

		response[blockchain] = append(response[blockchain], privateKeys...)
	}

	return &response, nil
}
