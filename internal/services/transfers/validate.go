package transfers

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

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
func (r *CreateTransferRequest) validate(conf *config.Config) error {
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

	if err := r.validatePurpose(); err != nil {
		return err
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

// normalize trims user input and lowercases the source tx hash,
// so the same deposit always hits the refund unique index
func (r *CreateTransferRequest) normalize() {
	if r.Purpose == "" {
		r.Purpose = constants.TransferPurposeDefault
	}

	r.TOTP = strings.TrimSpace(r.TOTP)

	if r.SourceTxHash != nil {
		hash := strings.ToLower(strings.TrimSpace(*r.SourceTxHash))
		r.SourceTxHash = &hash
	}
}

// feeFromAmount returns true when the network fee is subtracted from the amount: refunds of the native coin
func (r *CreateTransferRequest) feeFromAmount() bool {
	return r.Purpose == constants.TransferPurposeRefund && r.AssetIdentifier == r.Blockchain.GetAssetIdentifier()
}

// validatePurpose validates purpose-specific fields of the request
func (r *CreateTransferRequest) validatePurpose() error {
	if !r.Purpose.Valid() {
		return fmt.Errorf("invalid transfer purpose: %s", r.Purpose)
	}

	// external transfers must have an exact amount (transfers_external_amount_check)
	if r.Purpose != constants.TransferPurposeDefault && r.WholeAmount {
		return fmt.Errorf("whole amount is not allowed for %s transfers", r.Purpose)
	}

	// source tx hash is set for refunds only (transfers_refund_source_check)
	if r.Purpose == constants.TransferPurposeRefund {
		if r.SourceTxHash == nil || *r.SourceTxHash == "" {
			return fmt.Errorf("source tx hash is required for refund transfers")
		}

		if !isValidTxHash(r.Blockchain, *r.SourceTxHash) {
			return fmt.Errorf("invalid source tx hash %s", *r.SourceTxHash)
		}

		// deposit has a single recipient, the refund is sent from it
		if len(r.FromAddresses) != 1 {
			return fmt.Errorf("refund requires exactly one from address")
		}
	} else if r.SourceTxHash != nil {
		return fmt.Errorf("source tx hash is allowed for refund transfers only")
	}

	if r.TOTP == "" {
		if r.Purpose == constants.TransferPurposeWithdrawal {
			return fmt.Errorf("totp is required for withdrawal transfers: %w", rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorRequired))
		}
		return nil
	}

	if r.Purpose == constants.TransferPurposeDefault {
		return fmt.Errorf("totp is not allowed for default transfers")
	}

	if !totpRegexp.MatchString(r.TOTP) {
		return fmt.Errorf("invalid totp format: %w", rpccode.GetErrorByCode(rpccode.RPCCodeTwoFactorInvalid))
	}

	return nil
}

var (
	totpRegexp    = regexp.MustCompile(`^\d{6}$`)
	txHashRegexp  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	evmHashRegexp = regexp.MustCompile(`^0x[0-9a-f]{64}$`)
)

// isValidTxHash checks the normalized (lowercase) tx hash format for the blockchain
func isValidTxHash(blockchain wconstants.BlockchainType, hash string) bool {
	switch blockchain {
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return evmHashRegexp.MatchString(hash)
	default:
		return txHashRegexp.MatchString(hash)
	}
}

func (r *CreateTransferRequest) validateBitcoin() error {
	if r.Kind != nil {
		return fmt.Errorf("kind is not supported for the bitcoin blockchain")
	}

	if r.AssetIdentifier != btc.AssetIdentifier {
		return fmt.Errorf("asset identifier must be btc for the bitcoin blockchain")
	}

	return nil
}

func (r *CreateTransferRequest) validateLitecoin() error {
	if r.Kind != nil {
		return fmt.Errorf("kind is not supported for the litecoin blockchain")
	}

	if r.AssetIdentifier != ltc.AssetIdentifier {
		return fmt.Errorf("asset identifier must be btc for the litecoin blockchain")
	}

	return nil
}

func (r *CreateTransferRequest) validateTron(conf *config.Config) error {
	// refunds are paid by the hot wallet itself, all transfer kinds use the processing wallet
	if r.Purpose == constants.TransferPurposeRefund {
		if r.Kind != nil && *r.Kind != "" {
			return fmt.Errorf("kind is not supported for tron refunds")
		}
		return nil
	}

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

func (r *CreateTransferRequest) validateEVM() error {
	if r.Kind != nil {
		return fmt.Errorf("kind is not supported for the %s blockchain", r.Blockchain.String())
	}

	return nil
}
