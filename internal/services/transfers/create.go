package transfers

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfers"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
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

	walletFromType constants.WalletType

	walletToType constants.WalletType
}

// Create transfer
func (s *Service) Create(ctx context.Context, req CreateTransferRequest) (*models.Transfer, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	s.logger.Infof("received transfer request %s", string(reqBytes))

	if !s.config.Transfers.Enabled {
		return nil, fmt.Errorf("transfers service is disabled")
	}

	// validate request
	if err := req.validate(s.config); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
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

	// check to addresses and get wallet to type
	for idx, toAddress := range req.ToAddresses {
		if req.walletFromType == constants.WalletTypeProcessing {
			continue
		}

		checkResult, err := s.walletsSvc.CheckWallet(ctx, req.Blockchain, toAddress)
		if err != nil {
			return nil, fmt.Errorf("check wallet to for blockchain %s and address %s: %w", req.Blockchain, toAddress, err)
		}

		if checkResult.OwnerID != req.OwnerID {
			return nil, fmt.Errorf("invalid wallet owner %s", req.OwnerID)
		}

		if idx == 0 {
			req.walletToType = checkResult.WalletType
		} else if req.walletToType != checkResult.WalletType {
			return nil, fmt.Errorf("different wallet types in to addresses")
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

		// available wallet to types for transfer from hot wallet: cold and processing
		if req.walletFromType == constants.WalletTypeHot &&
			!slices.Contains([]constants.WalletType{constants.WalletTypeCold, constants.WalletTypeProcessing}, req.walletToType) {
			return nil, fmt.Errorf("invalid wallet to type %s", req.walletToType)
		}
	}

	// validate walletes
	switch req.Blockchain {
	case wconstants.BlockchainTypeBitcoin,
		wconstants.BlockchainTypeLitecoin,
		wconstants.BlockchainTypeBitcoinCash,
		wconstants.BlockchainTypeDogecoin:
		if err := s.processBTCLike(ctx, &req); err != nil {
			return nil, fmt.Errorf("process %s: %w", req.Blockchain, err)
		}
	case wconstants.BlockchainTypeTron:
		if err := s.processTron(ctx, &req); err != nil {
			return nil, fmt.Errorf("process tron: %w", err)
		}
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		if err := s.processEVM(ctx, &req); err != nil {
			return nil, fmt.Errorf("process evm: %w", err)
		}
	}

	if req.stateData == nil {
		req.stateData = make(map[string]any)
	}

	// set wallet to type
	req.stateData["wallet_to_type"] = req.walletToType

	createParams := repo_transfers.CreateParams{
		Status:          constants.TransferStatusNew,
		OwnerID:         owner.ID,
		ClientID:        owner.ClientID,
		RequestID:       req.RequestID,
		Blockchain:      req.Blockchain,
		FromAddresses:   req.FromAddresses,
		ToAddresses:     req.ToAddresses,
		WalletFromType:  req.walletFromType,
		Kind:            pgtypeutils.EncodeText(req.Kind),
		AssetIdentifier: req.AssetIdentifier,
		WholeAmount:     req.WholeAmount,
		Amount:          req.Amount,
		Fee:             req.Fee,
		FeeMax:          req.FeeMax,
		StateData:       req.stateData,
	}

	newTransfer, err := s.store.Transfers().Create(ctx, createParams)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, storecmn.ErrAlreadyExists
		}
		return nil, err
	}

	return newTransfer, nil
}
