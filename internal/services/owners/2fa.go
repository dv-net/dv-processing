package owners

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	issuerName    = "DVMerchant"
	otpSecretSize = 10
)

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

	// Validate as soon as we have it, since it's time based.
	// Also check for remaining request's fields to reduce store calls.
	if ok := totp.Validate(otp, owner.OtpSecret.String); !ok {
		return fmt.Errorf("failed to validate totp")
	}

	if owner.OtpConfirmed {
		return fmt.Errorf("owner already enabled two-factor authentication")
	}

	s.store.Cache().Owners().Delete(ownerID.String())

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

	if owner.OtpSecret.String == "" {
		return fmt.Errorf("owner has no secret")
	}

	// Validate as soon as we have it, since it's time based.
	// Also check for remaining request's fields to reduce store calls.
	if ok := totp.Validate(otpKey, owner.OtpSecret.String); !ok {
		return fmt.Errorf("failed to validate totp")
	}

	if !owner.OtpConfirmed {
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

	s.store.Cache().Owners().Delete(ownerID.String())

	return s.store.Owners().DisableTwoFactorAuth(ctx,
		pgtype.Text{
			String: newSecret.Secret(),
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

	if owner.OtpSecret.String == "" {
		return fmt.Errorf("owner has no secret")
	}

	if !owner.OtpConfirmed {
		return fmt.Errorf("owner has not confirmed two-factor authentication")
	}

	if ok := totp.Validate(token, owner.OtpSecret.String); !ok {
		return fmt.Errorf("failed to validate totp")
	}

	return nil
}
