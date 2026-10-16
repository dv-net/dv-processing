package transfers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/rpccode"
)

// Cancel cancels a scheduled transfer which has not been picked up for execution yet.
// onCanceled is called in the same db transaction, e.g. to create the status webhook.
func (s *Service) Cancel(
	ctx context.Context,
	ownerID uuid.UUID,
	requestID string,
	onCanceled func(ctx context.Context, transfer *models.Transfer, opts ...repos.Option) error,
) (*models.Transfer, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if requestID == "" {
		return nil, fmt.Errorf("request id is required")
	}

	var canceled *models.Transfer
	err := pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		canceled, err = s.store.Transfers(repos.WithTx(tx)).Cancel(ctx, ownerID, requestID)
		if err != nil {
			return convertScheduleError(err)
		}

		if onCanceled != nil {
			return onCanceled(ctx, canceled, repos.WithTx(tx))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return canceled, nil
}

// ExecuteNow confirms a scheduled transfer by the owner's TOTP and makes it due immediately.
func (s *Service) ExecuteNow(ctx context.Context, ownerID uuid.UUID, requestID string, totp string) (*models.Transfer, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if requestID == "" {
		return nil, fmt.Errorf("request id is required")
	}

	totp = strings.TrimSpace(totp)
	if totp == "" {
		return nil, rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorRequired)
	}

	if !totpRegexp.MatchString(totp) {
		return nil, fmt.Errorf("invalid totp format: %w", rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorInvalid))
	}

	if err := s.ownersSvc.ValidateTwoFactorToken(ctx, ownerID, totp); err != nil {
		return nil, fmt.Errorf("%w: %s", rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorInvalid), err.Error())
	}

	transfer, err := s.store.Transfers().ExecuteNow(ctx, ownerID, requestID)
	if err != nil {
		return nil, convertScheduleError(err)
	}

	return transfer, nil
}

// convertScheduleError maps "no rows" of conditional updates: the transfer does not exist,
// belongs to another owner, is not scheduled or has already been picked up for execution
func convertScheduleError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("transfer is not scheduled: %w", rpccode.GetErrorByCode(rpccode.RPCCodeTransferNotCancelable))
	}

	return err
}
