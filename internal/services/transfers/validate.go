package transfers

import (
	"fmt"
	"slices"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/avalidator"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

// validate request
func (r CreateTransferRequest) validate(conf *config.Config) error {
	if r.OwnerID == uuid.Nil {
		return fmt.Errorf("owner id is required")
	}

	if r.RequestID == "" {
		return fmt.Errorf("request id is required")
	}

	if !r.Blockchain.Valid() {
		return fmt.Errorf("invalid blockchain type: %s", r.Blockchain.String())
	}

	/* Check address from */
	switch r.Blockchain {
	case wconstants.BlockchainTypeBitcoin, wconstants.BlockchainTypeLitecoin, wconstants.BlockchainTypeBitcoinCash, wconstants.BlockchainTypeDogecoin:
		if len(r.FromAddresses) < 1 {
			return fmt.Errorf("from address is required")
		}

	default:
		if len(r.FromAddresses) != 1 {
			return fmt.Errorf("only one from address is supported")
		}
	}

	if len(r.ToAddresses) != 1 {
		return fmt.Errorf("only one to address is supported")
	}

	// check addresses for uniqueness
	for _, toAddress := range r.ToAddresses {
		if slices.Contains(r.FromAddresses, toAddress) {
			return fmt.Errorf("from and to addresses could be different")
		}
	}

	// check from addresses for uniqueness
	if dupl := lo.FindDuplicates(r.FromAddresses); len(dupl) > 0 {
		return fmt.Errorf("from addresses must be unique, duplicates: %v", dupl)
	}

	// check to addresses for uniqueness
	if dupl := lo.FindDuplicates(r.ToAddresses); len(dupl) > 0 {
		return fmt.Errorf("to addresses must be unique, duplicates: %v", dupl)
	}

	// validate addresses
	if err := avalidator.ValidateAddressesByBlockchain(
		append(r.FromAddresses, r.ToAddresses...),
		r.Blockchain.String(),
	); err != nil {
		return err
	}

	if r.AssetIdentifier == "" {
		return fmt.Errorf("asset identifier is required")
	}

	if len(r.AssetIdentifier) > 5 &&
		!avalidator.ValidateAddressByBlockchain(r.AssetIdentifier, r.Blockchain.String()) {
		return fmt.Errorf("invalid asset identifier: %s", r.AssetIdentifier)
	}

	if r.WholeAmount && r.Amount.Valid {
		return fmt.Errorf("whole amount and amount cannot be set at the same time")
	}

	if !r.WholeAmount && !r.Amount.Valid {
		return fmt.Errorf("amount is required")
	}

	if r.Amount.Valid && !r.Amount.Decimal.IsPositive() {
		return fmt.Errorf("amount must be greater than 0")
	}

	if r.Fee.Valid && !r.Fee.Decimal.IsPositive() {
		return fmt.Errorf("fee must be greater than 0")
	}

	if r.FeeMax.Valid && !r.FeeMax.Decimal.IsPositive() {
		return fmt.Errorf("max fee must be greater than 0")
	}

	switch r.Blockchain {
	case wconstants.BlockchainTypeBitcoin:
		return r.validateBitcoin()
	case wconstants.BlockchainTypeLitecoin:
		return r.validateLitecoin()
	case wconstants.BlockchainTypeTron:
		return r.validateTron(conf)
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return r.validateEVM()
	default:
		return nil
	}
}

func (r CreateTransferRequest) validateBitcoin() error {
	// if !r.WholeAmount {
	// 	return fmt.Errorf("currently only whole amount transfers are supported for bitcoin blockchain")
	// }

	if r.Kind != nil {
		return fmt.Errorf("kind is not supported for the bitcoin blockchain")
	}

	if r.AssetIdentifier != btc.AssetIdentifier {
		return fmt.Errorf("asset identifier must be btc for the bitcoin blockchain")
	}

	return nil
}

func (r CreateTransferRequest) validateLitecoin() error {
	// if !r.WholeAmount {
	// 	return fmt.Errorf("currently only whole amount transfers are supported for litecoin blockchain")
	// }

	if r.Kind != nil {
		return fmt.Errorf("kind is not supported for the litecoin blockchain")
	}

	if r.AssetIdentifier != ltc.AssetIdentifier {
		return fmt.Errorf("asset identifier must be btc for the litecoin blockchain")
	}

	return nil
}

func (r CreateTransferRequest) validateTron(conf *config.Config) error {
	if r.Kind == nil || *r.Kind == "" {
		return fmt.Errorf("kind is required for the tron transfers")
	}

	if *r.Kind == constants.TronTransferKindCloudDelegate.String() && !conf.ResourceManager.Enabled {
		return fmt.Errorf("resource manager is disabled %w", rpccode.GetErrorByCode(rpccode.RPCCodeServiceUnavailable))
	}

	if !constants.TronTransferKind(*r.Kind).Valid() {
		return fmt.Errorf("invalid transfer kind %s. available [burntrx] and [resources] kinds", *r.Kind)
	}

	return nil
}

func (r CreateTransferRequest) validateEVM() error {
	if r.Kind != nil {
		return fmt.Errorf("kind is not supported for the %s blockchain", r.Blockchain.String())
	}

	return nil
}
