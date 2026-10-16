package transfers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/dbutils/pgtypeutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
)

const refundSourceUniqIndex = "transfers_refund_source_uniq_idx"

// preparePurpose runs purpose-specific checks which require network or database calls
// and sets execute_after / two_factor_confirmed_at. It is called before the global lock.
func (s *Service) preparePurpose(ctx context.Context, req *CreateTransferRequest) error {
	switch req.Purpose {
	case constants.TransferPurposeDefault:
		return nil
	case constants.TransferPurposeWithdrawal:
		return s.prepareWithdrawal(ctx, req)
	case constants.TransferPurposeRefund:
		return s.prepareRefund(ctx, req)
	}

	return fmt.Errorf("unsupported transfer purpose %s", req.Purpose)
}

// prepareWithdrawal checks the owner's TOTP, withdrawals are executed immediately
func (s *Service) prepareWithdrawal(ctx context.Context, req *CreateTransferRequest) error {
	if err := s.validateTOTP(ctx, req); err != nil {
		return err
	}

	now := time.Now()
	req.twoFactorConfirmedAt = &now

	return nil
}

// prepareRefund checks the refunded deposit.
//
// With TOTP the refund is allowed to any external address and is executed immediately.
// Without TOTP the refund is allowed only back to the sender of the deposit and is delayed by the cooldown.
func (s *Service) prepareRefund(ctx context.Context, req *CreateTransferRequest) error {
	senders, err := s.verifyRefundSource(ctx, req)
	if err != nil {
		return err
	}

	exists, err := s.store.Transfers().ExistsActiveRefundBySource(ctx, req.OwnerID, req.Blockchain, pgtypeutils.EncodeText(req.SourceTxHash))
	if err != nil {
		return fmt.Errorf("check active refund: %w", err)
	}

	if exists {
		return fmt.Errorf("refund for deposit %s: %w", *req.SourceTxHash, rpccode.GetErrorByCode(rpccode.RPCCodeRefundAlreadyExists))
	}

	req.refundToSender = isRefundSender(req.Blockchain, senders, req.ToAddresses[0])

	// the check of a refund does not confirm it: TOTP and the destination rule are applied on creation
	if req.dryRun {
		return nil
	}

	now := time.Now()

	if req.TOTP != "" {
		if err := s.validateTOTP(ctx, req); err != nil {
			return err
		}
		req.twoFactorConfirmedAt = &now
		return nil
	}

	// a refund to a new address must be confirmed by the owner
	if !req.refundToSender {
		return fmt.Errorf(
			"refund to %s which is not the sender of the deposit: %w",
			req.ToAddresses[0], rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorRequired),
		)
	}

	executeAfter := now.Add(s.config.Transfers.RefundCooldown)
	req.executeAfter = &executeAfter

	return nil
}

// isRefundSender checks if the address is one of the deposit senders.
// EVM addresses are case-insensitive, other blockchains are compared exactly.
func isRefundSender(blockchain wconstants.BlockchainType, senders []string, address string) bool {
	for _, sender := range senders {
		if sender == address || (blockchain.IsEVM() && strings.EqualFold(sender, address)) {
			return true
		}
	}

	return false
}

// validateTOTP validates the owner's TOTP code
func (s *Service) validateTOTP(ctx context.Context, req *CreateTransferRequest) error {
	if req.TOTP == "" {
		return rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorRequired)
	}

	if err := s.ownersSvc.ValidateTwoFactorToken(ctx, req.OwnerID, req.TOTP); err != nil {
		return fmt.Errorf("%w: %s", rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorInvalid), err.Error())
	}

	return nil
}

// verifyRefundSource checks that the source tx is a confirmed deposit of the asset
// to the from address and the refund amount does not exceed the deposited amount.
// It returns the sender addresses of the deposit.
func (s *Service) verifyRefundSource(ctx context.Context, req *CreateTransferRequest) ([]string, error) {
	tx, err := s.eproxySvc.GetTransactionInfo(ctx, req.Blockchain, *req.SourceTxHash)
	if err != nil {
		return nil, fmt.Errorf("get source tx %s: %w", *req.SourceTxHash, err)
	}

	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", rpccode.GetErrorByCode(rpccode.RPCCodeRefundSourceInvalid), fmt.Sprintf(format, args...))
	}

	if tx.Confirmations < constants.GetMinConfirmations(req.Blockchain) {
		return nil, invalid("source tx %s is not confirmed: %d confirmations", tx.Hash, tx.Confirmations)
	}

	fromAddress := req.FromAddresses[0]
	deposited := decimal.Zero
	senders := make([]string, 0, 2)
	if tx.GetAddressFrom() != "" {
		senders = append(senders, tx.GetAddressFrom())
	}
	for _, ev := range tx.GetEvents() {
		if ev.Type == nil || *ev.Type != transactionsv2.EventType_EVENT_TYPE_TRANSFER {
			continue
		}
		if ev.Status != nil && *ev.Status != transactionsv2.EventStatus_EVENT_STATUS_SUCCESS {
			continue
		}
		if !strings.EqualFold(ev.GetAddressTo(), fromAddress) {
			continue
		}
		if !strings.EqualFold(ev.GetAssetIdentifier(), req.AssetIdentifier) {
			continue
		}

		value, err := decimal.NewFromString(ev.GetValue())
		if err != nil || !value.IsPositive() {
			continue
		}
		deposited = deposited.Add(value)

		if ev.GetAddressFrom() != "" {
			senders = append(senders, ev.GetAddressFrom())
		}
	}

	if !deposited.IsPositive() {
		return nil, invalid("source tx %s has no deposit of %s to %s", tx.Hash, req.AssetIdentifier, fromAddress)
	}

	if req.Amount.Decimal.GreaterThan(deposited) {
		return nil, invalid("refund amount %s exceeds deposited amount %s", req.Amount.Decimal, deposited)
	}

	return senders, nil
}

// resolveWalletTo sets the destination wallet type. An address which does not belong
// to any wallet is external and allowed for withdrawals and refunds only.
func (s *Service) resolveWalletTo(ctx context.Context, req *CreateTransferRequest) error {
	externalAllowed := req.Purpose == constants.TransferPurposeWithdrawal || req.Purpose == constants.TransferPurposeRefund

	for idx, toAddress := range req.ToAddresses {
		walletType := constants.WalletTypeExternal

		checkResult, err := s.walletsSvc.CheckWallet(ctx, req.Blockchain, toAddress)
		switch {
		case err == nil:
			if checkResult.OwnerID != req.OwnerID {
				return fmt.Errorf("invalid wallet owner %s", req.OwnerID)
			}
			walletType = checkResult.WalletType
		case errors.Is(err, wallets.ErrAddressNotFound) && externalAllowed:
		default:
			return fmt.Errorf("check wallet to for blockchain %s and address %s: %w", req.Blockchain, toAddress, err)
		}

		// withdrawals and refunds to the owner's wallets must use regular transfers
		if externalAllowed && walletType != constants.WalletTypeExternal {
			return fmt.Errorf("%s transfer to the owner's %s wallet is not allowed", req.Purpose, walletType)
		}

		if idx == 0 {
			req.walletToType = walletType
		} else if req.walletToType != walletType {
			return fmt.Errorf("different wallet types in to addresses")
		}
	}

	return nil
}

// convertCreateError maps unique violations to domain errors
func convertCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == refundSourceUniqIndex {
			return rpccode.GetErrorByCode(rpccode.RPCCodeRefundAlreadyExists)
		}
		return storecmn.ErrAlreadyExists
	}

	if strings.Contains(err.Error(), "unique") {
		return storecmn.ErrAlreadyExists
	}

	return err
}

func encodeTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

type CheckRefundResult struct {
	// EstimatedFee is nil for btc like blockchains: the fee is calculated only on sending
	EstimatedFee  *decimal.Decimal
	FeeFromAmount bool
	ToSender      bool
}

// CheckRefund runs all checks of a refund without creating it: the deposit, the destination
// and that the hot wallet pays the network fee by itself. TOTP is not checked.
func (s *Service) CheckRefund(ctx context.Context, req CreateTransferRequest) (*CheckRefundResult, error) {
	req.Purpose = constants.TransferPurposeRefund
	req.TOTP = ""
	req.dryRun = true
	// request id is required by the validation, the transfer is not created
	if req.RequestID == "" {
		req.RequestID = uuid.NewString()
	}

	req.normalize()

	if !s.config.Transfers.Enabled {
		return nil, fmt.Errorf("transfers service is disabled")
	}

	if err := req.validate(s.config); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := s.preparePurpose(ctx, &req); err != nil {
		return nil, fmt.Errorf("prepare refund: %w", err)
	}

	if _, err := s.checkTransfer(ctx, &req); err != nil {
		return nil, err
	}

	return &CheckRefundResult{
		EstimatedFee:  req.refundEstimatedFee,
		FeeFromAmount: req.feeFromAmount(),
		ToSender:      req.refundToSender,
	}, nil
}
