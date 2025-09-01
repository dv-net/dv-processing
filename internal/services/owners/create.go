package owners

import (
	"context"
	"fmt"
	"strings"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_owners"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/dbutils/pgtypeutils"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/go-bip39"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type CreateParams struct {
	ClientID   uuid.UUID `json:"client_id" validate:"required,uuid"`
	ExternalID string    `json:"external_id" validate:"required"`
	Mnemonic   string    `json:"mnemonic" validate:"required"`
}

// Create creates a new owner.
func (s *Service) Create(ctx context.Context, params CreateParams) (*models.Owner, error) {
	if err := s.validator.Struct(params); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// check if clinet clientExists
	clientExists, err := s.store.Clients().ExistsByID(ctx, params.ClientID)
	if err != nil {
		return nil, fmt.Errorf("check client exists: %w", err)
	}
	if !clientExists {
		return nil, ErrClientNotFound
	}

	ownerExistsByExternalID, err := s.store.Owners().ExistsByExternalID(ctx, params.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("check external id: %w", err)
	}

	// if externalID is already used, return an error
	if ownerExistsByExternalID {
		return nil, ErrExternalIDExists
	}

	if !bip39.IsMnemonicValid(params.Mnemonic) {
		return nil, fmt.Errorf("mnemonic is invalid")
	}

	// entropy, err := bip39.NewEntropy(256) //nolint:mnd
	// if err != nil {
	// 	return nil, fmt.Errorf("generate entropy: %w", err)
	// }

	// mnemonic, err := bip39.NewMnemonic(entropy)
	// if err != nil {
	// 	return nil, fmt.Errorf("generate mnemonic: %w", err)
	// }

	// passphrase, err := utils.SpecialKey(16) //nolint:mnd
	// if err != nil {
	// 	return nil, fmt.Errorf("generate pass phrase: %w", err)
	// }

	createParams := repo_owners.CreateParams{
		ClientID:   params.ClientID,
		ExternalID: params.ExternalID,
		Mnemonic:   params.Mnemonic,
		// PassPhrase: pgtypeutils.EncodeText(&passphrase),
	}

	if err := s.validator.Struct(createParams); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	var owner *models.Owner
	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		var err error

		// create owner
		owner, err = s.store.Owners(repos.WithTx(dbTx)).Create(ctx, createParams)
		if err != nil {
			return fmt.Errorf("create owner: %w", err)
		}

		var encryptedMnemonic string

		// update mnemonic
		if s.config.IsEnabledSeedEncryption() {
			encryptedMnemonic, err = encryption.Encrypt(params.Mnemonic, owner.ID.String())
			if err != nil {
				return fmt.Errorf("encrypt mnemonic: %w", err)
			}

			if err := s.store.Owners(repos.WithTx(dbTx)).UpdateMnemonic(ctx, owner.ID, encryptedMnemonic); err != nil {
				return fmt.Errorf("update mnemonic: %w", err)
			}
		}

		totpSecret, err := totp.Generate(
			totp.GenerateOpts{
				Issuer:      issuerName,
				AccountName: owner.ID.String(),
				Algorithm:   otp.AlgorithmSHA1,
				SecretSize:  otpSecretSize,
			})
		if err != nil {
			return fmt.Errorf("generate totp secret: %w", err)
		}

		// set otp secret
		if err := s.store.Owners(repos.WithTx(dbTx)).SetOTPSecret(ctx, owner.ID, pgtypeutils.EncodeText(
			utils.Pointer(totpSecret.Secret()),
		)); err != nil {
			return fmt.Errorf("set otp secret: %w", err)
		}

		// create processing wallets for all available blockchains
		for _, blockchain := range wconstants.AllBlockchains {
			createParams := wallets.CreateProcessingWalletParams{
				OwnerID:    owner.ID,
				Blockchain: blockchain,
				Mnemonic:   encryptedMnemonic,
				// Passphrase: passphrase,
			}

			if _, err := s.walletsSvc.Processing().Create(ctx, createParams, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("create processing wallet: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, storecmn.ErrAlreadyExists
		}
		return nil, err
	}

	return owner, nil
}
