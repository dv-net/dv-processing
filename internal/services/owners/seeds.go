package owners

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type GetSeedsResponse struct {
	Mnemonic   string
	PassPhrase string
}

// GetSeeds returns the mnemonic and passphrase for the owner.
func (s *Service) GetSeeds(ctx context.Context, ownerID uuid.UUID, otp string) (*GetSeedsResponse, error) {
	if otp == "" {
		return nil, storecmn.ErrEmptyOTP
	}

	owner, err := s.GetByID(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	if owner.Mnemonic == "" {
		return nil, fmt.Errorf("owner has no mnemonic")
	}

	if !s.getOTPConfirmed(owner) {
		return nil, fmt.Errorf("two-factor authenticator is disabled")
	}

	// get 2FA secret from either new or legacy format
	if ok := s.ValidateTwoFactorToken(ctx, owner.ID, otp); ok != nil {
		return nil, fmt.Errorf("validate otp: %w", ok)
	}

	// if owner.PassPhrase.String == "" {
	// 	return nil, fmt.Errorf("owner has no passphrase")
	// }

	mnemonic := owner.Mnemonic
	if s.config.IsEnabledSeedEncryption() {
		mnemonic, err = encryption.Decrypt(mnemonic, owner.ID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	return &GetSeedsResponse{
		Mnemonic:   mnemonic,
		PassPhrase: owner.PassPhrase.String,
	}, nil
}

func (s *Service) EncryptSeedsForAllOwners(ctx context.Context) error {
	owners, err := s.store.Owners().GetAll(ctx)
	if err != nil {
		return err
	}

	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		for _, owner := range owners {
			if encryption.IsEncrypted(owner.Mnemonic) {
				continue
			}

			mnemonic, err := encryption.Encrypt(owner.Mnemonic, owner.ID.String())
			if err != nil {
				return err
			}

			if err := s.store.Owners(repos.WithTx(tx)).UpdateMnemonic(ctx, owner.ID, mnemonic); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (s *Service) DecryptSeedsForAllOwners(ctx context.Context) error {
	owners, err := s.store.Owners().GetAll(ctx)
	if err != nil {
		return err
	}

	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		for _, owner := range owners {
			if !encryption.IsEncrypted(owner.Mnemonic) {
				continue
			}

			mnemonic, err := encryption.Decrypt(owner.Mnemonic, owner.ID.String())
			if err != nil {
				return err
			}

			if err := s.store.Owners(repos.WithTx(tx)).UpdateMnemonic(ctx, owner.ID, mnemonic); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}
