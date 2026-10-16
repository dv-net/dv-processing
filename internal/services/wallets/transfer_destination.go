package wallets

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
)

// ValidateTransferDestination checks the destination of a transfer from a hot wallet.
//
// Regular transfers are allowed to the owner's cold and processing wallets only.
// Withdrawals and refunds are allowed to external addresses, which do not belong to any wallet,
// and must be confirmed by 2FA or, for refunds, delayed by the cooldown.
func (s *Service) ValidateTransferDestination(ctx context.Context, transfer *models.Transfer) error {
	if transfer.WalletFromType != constants.WalletTypeHot {
		return nil
	}

	if transfer.WalletToType != nil && *transfer.WalletToType == constants.WalletTypeExternal {
		return s.validateExternalDestination(ctx, transfer)
	}

	checkResult, err := s.CheckWallet(ctx, transfer.Blockchain, transfer.GetToAddress())
	if err != nil {
		return fmt.Errorf("check wallet: %w", err)
	}

	if !slices.Contains([]constants.WalletType{constants.WalletTypeCold, constants.WalletTypeProcessing}, checkResult.WalletType) {
		return fmt.Errorf("invalid wallet type %s", checkResult.WalletType)
	}

	if checkResult.OwnerID != transfer.OwnerID {
		return fmt.Errorf("invalid wallet owner %s", checkResult.OwnerID)
	}

	return nil
}

func (s *Service) validateExternalDestination(ctx context.Context, transfer *models.Transfer) error {
	switch transfer.Purpose {
	case constants.TransferPurposeWithdrawal:
		if !transfer.TwoFactorConfirmedAt.Valid {
			return fmt.Errorf("withdrawal to external address is not confirmed by 2fa")
		}
	case constants.TransferPurposeRefund:
		if !transfer.SourceTxHash.Valid || transfer.SourceTxHash.String == "" {
			return fmt.Errorf("refund source tx hash is empty")
		}
		if !transfer.TwoFactorConfirmedAt.Valid && !transfer.ExecuteAfter.Valid {
			return fmt.Errorf("refund to external address is neither confirmed by 2fa nor delayed")
		}
	default:
		return fmt.Errorf("transfer with purpose %s to external address is not allowed", transfer.Purpose)
	}

	// the address must still not belong to any wallet
	checkResult, err := s.CheckWallet(ctx, transfer.Blockchain, transfer.GetToAddress())
	if err == nil {
		return fmt.Errorf("external address belongs to %s wallet", checkResult.WalletType)
	}

	if !errors.Is(err, ErrAddressNotFound) {
		return fmt.Errorf("check wallet: %w", err)
	}

	return nil
}
