package transfers

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfers"
	"github.com/dv-net/dv-processing/pkg/dbutils/pgtypeutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

type CreateTransferRequest struct {
	OwnerID         uuid.UUID                 `json:"owner_id"`
	RequestID       string                    `json:"request_id"`
	Blockchain      wconstants.BlockchainType `json:"blockchain"`
	FromAddresses   []string                  `json:"from_addresses"`
	ToAddresses     []string                  `json:"to_addresses"`
	AssetIdentifier string                    `json:"asset_identifier"`
	Kind            *string                   `json:"kind"`
	WholeAmount     bool                      `json:"whole_amount"`
	Amount          decimal.NullDecimal       `json:"amount"`
	Fee             decimal.NullDecimal       `json:"fee"`
	FeeMax          decimal.NullDecimal       `json:"fee_max"`

	stateData map[string]any

	Purpose      constants.TransferPurpose `json:"purpose"`
	SourceTxHash *string                   `json:"source_tx_hash"`
	TOTP         string                    `json:"-"`

	// refundToSender is set when the refund destination is the sender of the deposit
	refundToSender bool
	// refundEstimatedFee is the network fee paid by the hot wallet for the refund, in the native coin
	refundEstimatedFee *decimal.Decimal
	// dryRun runs all checks of the transfer without creating it and without TOTP checks
	dryRun bool

	walletFromType       constants.WalletType
	walletToType         constants.WalletType
	executeAfter         *time.Time
	twoFactorConfirmedAt *time.Time
}

// Create transfer
func (s *Service) Create(ctx context.Context, req CreateTransferRequest) (*models.Transfer, error) {
	req.normalize()

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	s.logger.Debugf("received transfer request %s", string(reqBytes))

	if !s.config.Transfers.Enabled {
		return nil, fmt.Errorf("transfers service is disabled")
	}

	// validate request
	if err := req.validate(s.config); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// network and TOTP checks are done before the lock to not block other transfers
	if err := s.preparePurpose(ctx, &req); err != nil {
		return nil, fmt.Errorf("prepare %s transfer: %w", req.Purpose, err)
	}

	s.locker.L.Lock()
	for s.isLocked.Load() {
		s.locker.Wait()
	}
	s.isLocked.Store(true)
	s.locker.L.Unlock()

	defer func() {
		s.locker.L.Lock()
		s.isLocked.Store(false)
		s.locker.Signal()
		s.locker.L.Unlock()
	}()

	owner, err := s.checkTransfer(ctx, &req)
	if err != nil {
		return nil, err
	}

	if req.stateData == nil {
		req.stateData = make(map[string]any)
	}

	// set wallet to type
	req.stateData["wallet_to_type"] = req.walletToType

	if req.Purpose == constants.TransferPurposeRefund {
		req.stateData["refund_to_sender"] = req.refundToSender
	}

	createParams := repo_transfers.CreateParams{
		Status:               constants.TransferStatusNew,
		OwnerID:              owner.ID,
		ClientID:             owner.ClientID,
		RequestID:            req.RequestID,
		Blockchain:           req.Blockchain,
		FromAddresses:        req.FromAddresses,
		ToAddresses:          req.ToAddresses,
		WalletFromType:       req.walletFromType,
		Kind:                 pgtypeutils.EncodeText(req.Kind),
		AssetIdentifier:      req.AssetIdentifier,
		WholeAmount:          req.WholeAmount,
		Amount:               req.Amount,
		Fee:                  req.Fee,
		FeeMax:               req.FeeMax,
		StateData:            req.stateData,
		Purpose:              req.Purpose,
		SourceTxHash:         pgtypeutils.EncodeText(req.SourceTxHash),
		ExecuteAfter:         encodeTimestamptz(req.executeAfter),
		TwoFactorConfirmedAt: encodeTimestamptz(req.twoFactorConfirmedAt),
	}

	// wallet to type is unknown for transfers from processing wallets
	if req.walletToType != "" {
		createParams.WalletToType = &req.walletToType
	}

	newTransfer, err := s.store.Transfers().Create(ctx, createParams)
	if err != nil {
		return nil, convertCreateError(err)
	}

	return newTransfer, nil
}

// checkTransfer checks the owner, wallets and balances of the transfer by blockchain
func (s *Service) checkTransfer(ctx context.Context, req *CreateTransferRequest) (*models.Owner, error) {
	// check owner
	owner, err := s.store.Owners().GetByID(ctx, req.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	// check from addresses and get wallet from type
	for idx, fromAddress := range req.FromAddresses {
		// get wallet data
		checkWalletResult, err := s.walletsSvc.CheckWallet(ctx, req.Blockchain, fromAddress)
		if err != nil {
			return nil, fmt.Errorf("check wallet from for blockchain %s and address %s: %w", req.Blockchain, fromAddress, err)
		}

		// check wallet owner id
		if checkWalletResult.OwnerID != req.OwnerID {
			return nil, fmt.Errorf("invalid wallet owner %s", req.OwnerID)
		}

		if idx == 0 {
			// set wallet type
			req.walletFromType = checkWalletResult.WalletType
		} else if req.walletFromType != checkWalletResult.WalletType {
			// check wallet type for all from addresses in the request is the same
			return nil, fmt.Errorf("different wallet types in from addresses")
		}
	}

	// withdrawals and refunds are sent from hot wallets only
	if req.Purpose != constants.TransferPurposeDefault && req.walletFromType != constants.WalletTypeHot {
		return nil, fmt.Errorf("%s transfer is allowed from hot wallet only, got %s", req.Purpose, req.walletFromType)
	}

	// check to addresses and get wallet to type
	if req.walletFromType != constants.WalletTypeProcessing {
		if err := s.resolveWalletTo(ctx, req); err != nil {
			return nil, err
		}
	}

	// check wallet from type
	{
		// validate wallet type
		if !req.walletFromType.Valid() {
			return nil, fmt.Errorf("invalid wallet from type %s", req.walletFromType)
		}

		// available wallet from types for transfer: hot and processing
		if !slices.Contains([]constants.WalletType{constants.WalletTypeHot, constants.WalletTypeProcessing}, req.walletFromType) {
			return nil, fmt.Errorf("invalid wallet from type for transfer (%s)", req.walletFromType)
		}
	}

	// check wallet to type
	if req.walletFromType != constants.WalletTypeProcessing {
		// validate wallet type
		if !req.walletToType.Valid() {
			return nil, fmt.Errorf("invalid wallet to type %s", req.walletToType)
		}

		// available wallet to types for transfer from hot wallet: cold, processing
		// and external for withdrawals and refunds (checked in resolveWalletTo)
		if req.walletFromType == constants.WalletTypeHot &&
			!slices.Contains([]constants.WalletType{constants.WalletTypeCold, constants.WalletTypeProcessing, constants.WalletTypeExternal}, req.walletToType) {
			return nil, fmt.Errorf("invalid wallet to type %s", req.walletToType)
		}
	}

	// validate walletes
	switch req.Blockchain {
	case wconstants.BlockchainTypeBitcoin,
		wconstants.BlockchainTypeLitecoin,
		wconstants.BlockchainTypeBitcoinCash,
		wconstants.BlockchainTypeDogecoin:
		if err := s.processBTCLike(ctx, req); err != nil {
			return nil, fmt.Errorf("process %s: %w", req.Blockchain, err)
		}
	case wconstants.BlockchainTypeTron:
		if err := s.processTron(ctx, req); err != nil {
			return nil, fmt.Errorf("process tron: %w", err)
		}
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		if err := s.processEVM(ctx, req); err != nil {
			return nil, fmt.Errorf("process evm: %w", err)
		}
	}

	return owner, nil
}
