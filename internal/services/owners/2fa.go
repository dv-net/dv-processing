package owners

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	issuerName    = "DVMerchant"
	otpSecretSize = 10
)

type OTPData struct {
	OtpSecret    string `json:"otp_secret"`
	OtpConfirmed bool   `json:"otp_confirmed"`
}

func (s *Service) getOTPSecret(owner *models.Owner) (string, error) {
	// Try new otp_data format first
	if owner.OtpData.Valid && owner.OtpData.String != "" {
		otpDataStr := owner.OtpData.String

		// Decrypt if encrypted
		if encryption.IsEncrypted(owner.OtpData.String) {
			var err error
			otpDataStr, err = encryption.Decrypt(owner.OtpData.String, owner.ID.String())
			if err != nil {
				return "", fmt.Errorf("decrypt otp data: %w", err)
			}
		}

		var otpData OTPData
		if err := json.Unmarshal([]byte(otpDataStr), &otpData); err != nil {
			return "", fmt.Errorf("unmarshal otp data: %w", err)
		}

		return otpData.OtpSecret, nil
	}

	if owner.OtpSecret.Valid && owner.OtpSecret.String != "" {
		otpSecret, err := encryption.Decrypt(owner.OtpSecret.String, owner.ID.String())
		if err != nil {
			return "", fmt.Errorf("decrypt legacy otp secret: %w", err)
		}
		return otpSecret, nil
	}

	return "", fmt.Errorf("no otp secret found")
}

// getOTPConfirmed extracts the OTP confirmed status from either otp_data (new format) or otp_confirmed (legacy format)
func (s *Service) getOTPConfirmed(owner *models.Owner) bool {
	// Try new otp_data format first
	if owner.OtpData.Valid && owner.OtpData.String != "" {
		otpDataStr := owner.OtpData.String

		// Decrypt if encrypted
		if encryption.IsEncrypted(owner.OtpData.String) {
			var err error
			otpDataStr, err = encryption.Decrypt(owner.OtpData.String, owner.ID.String())
			if err != nil {
				return owner.OtpConfirmed
			}
		}

		var otpData OTPData
		if err := json.Unmarshal([]byte(otpDataStr), &otpData); err != nil {
			return owner.OtpConfirmed
		}

		return otpData.OtpConfirmed
	}

	return owner.OtpConfirmed
}

// ConfirmTwoFactorAuth confirms two-factor authentication for the owner.
func (s *Service) ConfirmTwoFactorAuth(ctx context.Context, ownerID uuid.UUID, otp string) error {
	if ownerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if otp == "" {
		return storecmn.ErrEmptyOTP
	}

	owner, err := s.GetByID(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("get owner: %w", err)
	}

	otpSecret, err := s.getOTPSecret(owner)
	if err != nil {
		return fmt.Errorf("get otp secret: %w", err)
	}

	// Validate as soon as we have it, since it's time based.
	// Also check for remaining request's fields to reduce store calls.
	if ok := totp.Validate(otp, otpSecret); !ok {
		return fmt.Errorf("failed to validate totp")
	}

	if s.getOTPConfirmed(owner) {
		return fmt.Errorf("owner already enabled two-factor authentication")
	}

	s.store.Cache().Owners().Delete(ownerID.String())

	// Create OTP data
	otpData := OTPData{
		OtpSecret:    otpSecret,
		OtpConfirmed: true,
	}
	otpDataStr, err := json.Marshal(otpData)
	if err != nil {
		return fmt.Errorf("marshal otp data: %w", err)
	}
	// Always encrypt new OTP data
	encryptedOtpData, err := encryption.Encrypt(string(otpDataStr), owner.ID.String())
	if err != nil {
		return fmt.Errorf("encrypt otp data: %w", err)
	}
	if err := s.store.Owners().SetOTPData(ctx, owner.ID, pgtype.Text{
		String: encryptedOtpData,
		Valid:  true,
	}); err != nil {
		return fmt.Errorf("set otp data: %w", err)
	}

	return s.store.Owners().ConfirmTwoFactorAuth(ctx, ownerID)
}

// DisableTwoFactorAuth disables two-factor authentication for the owner.
func (s *Service) DisableTwoFactorAuth(ctx context.Context, ownerID uuid.UUID, otpKey string) error {
	if ownerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if otpKey == "" {
		return storecmn.ErrEmptyOTP
	}

	owner, err := s.GetByID(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("get owner: %w", err)
	}

	// get 2FA secret from either new or legacy format
	otpSecret, err := s.getOTPSecret(owner)
	if err != nil {
		return fmt.Errorf("get otp secret: %w", err)
	}

	// Validate as soon as we have it, since it's time based.
	// Also check for remaining request's fields to reduce store calls.
	if ok := totp.Validate(otpKey, otpSecret); !ok {
		return fmt.Errorf("failed to validate totp")
	}

	if !s.getOTPConfirmed(owner) {
		return fmt.Errorf("owner already disabled two-factor authentication")
	}

	newSecret, err := totp.Generate(
		totp.GenerateOpts{
			Issuer:      issuerName,
			AccountName: owner.ID.String(),
			Algorithm:   otp.AlgorithmSHA1,
			SecretSize:  otpSecretSize,
		})
	if err != nil {
		return fmt.Errorf("generate totp secret: %w", err)
	}

	finalNewSecret := newSecret.Secret()

	s.store.Cache().Owners().Delete(ownerID.String())

	// Create new OTP data
	otpData := OTPData{
		OtpSecret:    finalNewSecret,
		OtpConfirmed: false,
	}
	otpDataStr, err := json.Marshal(otpData)
	if err != nil {
		return fmt.Errorf("marshal otp data: %w", err)
	}

	// Always encrypt new OTP data
	encryptedOtpData, err := encryption.Encrypt(string(otpDataStr), owner.ID.String())
	if err != nil {
		return fmt.Errorf("encrypt otp data: %w", err)
	}
	if err := s.store.Owners().SetOTPData(ctx, owner.ID, pgtype.Text{
		String: encryptedOtpData,
		Valid:  true,
	}); err != nil {
		return fmt.Errorf("set otp data: %w", err)
	}

	return s.store.Owners().DisableTwoFactorAuth(ctx,
		pgtype.Text{
			String: finalNewSecret,
			Valid:  true,
		}, ownerID,
	)
}

type GetTwoFactorAuthDataResponse struct {
	Secret      *string
	IsConfirmed bool
}

// GetTwoFactorAuthData returns the two-factor authentication data for the owner.
func (s *Service) GetTwoFactorAuthData(ctx context.Context, ownerID uuid.UUID) (*GetTwoFactorAuthDataResponse, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	owner, err := s.GetByID(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	if owner.OtpSecret.String == "" {
		return nil, fmt.Errorf("owner has no secret")
	}

	if owner.OtpConfirmed {
		return &GetTwoFactorAuthDataResponse{
			IsConfirmed: owner.OtpConfirmed,
		}, nil
	}

	res := &GetTwoFactorAuthDataResponse{
		IsConfirmed: owner.OtpConfirmed,
	}

	if owner.OtpSecret.String != "" {
		res.Secret = &owner.OtpSecret.String
	}

	return res, nil
}

// ValidateTwoFactorToken validates the two-factor token for the owner.
func (s *Service) ValidateTwoFactorToken(ctx context.Context, ownerID uuid.UUID, token string) error {
	if ownerID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if token == "" {
		return storecmn.ErrEmptyOTP
	}

	owner, err := s.GetByID(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("get owner: %w", err)
	}

	if !s.getOTPConfirmed(owner) {
		return fmt.Errorf("owner has not confirmed two-factor authentication")
	}

	// get 2FA secret from either new or legacy format
	otpSecret, err := s.getOTPSecret(owner)
	if err != nil {
		return fmt.Errorf("get otp secret: %w", err)
	}

	if ok := totp.Validate(token, otpSecret); !ok {
		return fmt.Errorf("failed to validate totp")
	}

	return nil
}

func (s *Service) EncryptOTPDataForAllOwners(ctx context.Context) error {
	owners, err := s.store.Owners().GetAll(ctx)
	if err != nil {
		return err
	}

	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		for _, owner := range owners {
			if owner.OtpData.String == "" || encryption.IsEncrypted(owner.OtpData.String) {
				continue
			}

			encryptedData, err := encryption.Encrypt(owner.OtpData.String, owner.ID.String())
			if err != nil {
				return err
			}

			if err := s.store.Owners(repos.WithTx(tx)).SetOTPData(ctx, owner.ID, pgtype.Text{
				String: encryptedData,
				Valid:  true,
			}); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (s *Service) DecryptOTPDataForAllOwners(ctx context.Context) error {
	owners, err := s.store.Owners().GetAll(ctx)
	if err != nil {
		return err
	}

	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		for _, owner := range owners {
			if owner.OtpData.String == "" || !encryption.IsEncrypted(owner.OtpData.String) {
				continue
			}

			decryptedData, err := encryption.Decrypt(owner.OtpData.String, owner.ID.String())
			if err != nil {
				return err
			}

			if err := s.store.Owners(repos.WithTx(tx)).SetOTPData(ctx, owner.ID, pgtype.Text{
				String: decryptedData,
				Valid:  true,
			}); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}
