package transfers

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

// processLitecoin handle transfer request for btc like blockchains
func (s *Service) processBTCLike(ctx context.Context, req *CreateTransferRequest) error {
	var (
		feeEstimator     btcLikeFeeEstimator
		configFeePerByte decimal.Decimal
		network          string
	)

	switch req.Blockchain {
	case wconstants.BlockchainTypeBitcoin:
		if !s.config.Blockchain.Bitcoin.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
		feeEstimator = s.blockchains.Bitcoin
		configFeePerByte = decimal.NewFromInt(s.config.Blockchain.Bitcoin.Attributes.FeePerByte)
		network = s.config.Blockchain.Bitcoin.Network
	case wconstants.BlockchainTypeLitecoin:
		if !s.config.Blockchain.Litecoin.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
		feeEstimator = s.blockchains.Litecoin
		configFeePerByte = decimal.NewFromInt(s.config.Blockchain.Litecoin.Attributes.FeePerByte)
		network = s.config.Blockchain.Litecoin.Network
	case wconstants.BlockchainTypeBitcoinCash:
		if !s.config.Blockchain.BitcoinCash.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
		feeEstimator = s.blockchains.BitcoinCash
		configFeePerByte = decimal.NewFromInt(s.config.Blockchain.BitcoinCash.Attributes.FeePerByte)
		network = s.config.Blockchain.BitcoinCash.Network
	case wconstants.BlockchainTypeDogecoin:
		if !s.config.Blockchain.Dogecoin.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
		feeEstimator = s.blockchains.Dogecoin
		configFeePerByte = decimal.NewFromInt(s.config.Blockchain.Dogecoin.Attributes.FeePerByte)
		network = s.config.Blockchain.Dogecoin.Network
	default:
		return fmt.Errorf("unsupported blockchain: %s", req.Blockchain)
	}

	// change of a transfer with amount is sent back to the single from address
	if !req.WholeAmount && len(req.FromAddresses) != 1 {
		return fmt.Errorf("only one from address is supported for transfer with amount")
	}

	// check fee
	if req.walletFromType == constants.WalletTypeHot && network != "testnet" &&
		(req.FeeMax.Valid || configFeePerByte.IsPositive()) {
		if err := checkBTCLikeFeePerByte(req, feeEstimator, configFeePerByte); err != nil {
			return err
		}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(10)

	// check balances
	for _, fromAddress := range req.FromAddresses {
		eg.Go(func() error {
			balance, err := s.eproxySvc.AddressBalance(egCtx, fromAddress, req.AssetIdentifier, req.Blockchain)
			if err != nil {
				return fmt.Errorf("get balance: %w", err)
			}

			// TODO: edit this condition
			if !req.WholeAmount && balance.LessThanOrEqual(req.Amount.Decimal) {
				return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), req.Amount.Decimal, balance)
			}

			if req.WholeAmount && !balance.IsPositive() {
				return fmt.Errorf("%w for transfer with whole amount, available: 0", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance))
			}

			// check active transfers with the same from address
			transfersInProcess, err := s.store.Transfers().Find(ctx, FindParams{
				StatusesIn:  []string{constants.TransferStatusProcessing.String()},
				FromAddress: &fromAddress,
				Blockchain:  &req.Blockchain,
				Limit:       utils.Pointer(1),
			})
			if err != nil {
				return fmt.Errorf("find transfers: %w", err)
			}

			if len(transfersInProcess) > 0 {
				return fmt.Errorf("%s: %w", fromAddress, rpccode.GetErrorByCode(rpccode.RPCCodeAddressIsTaken))
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("check balances on from addresses: %w", err)
	}

	return nil
}

type btcLikeFeeEstimator interface {
	EstimateFeePerByte() (decimal.Decimal, error)
}

// checkBTCLikeFeePerByte checks that the fee per byte for the transfer does not exceed
// the max fee from the request or the configured fee per byte.
// It mirrors the fee per byte selection of the btc like transfer FSMs.
func checkBTCLikeFeePerByte(req *CreateTransferRequest, feeEstimator btcLikeFeeEstimator, configFeePerByte decimal.Decimal) error {
	feePerByte := req.Fee.Decimal
	if !req.Fee.Valid || !feePerByte.IsPositive() {
		estimated, err := feeEstimator.EstimateFeePerByte()
		if err != nil {
			return fmt.Errorf("estimate fee per byte: %w", err)
		}

		// node has no estimate yet, the configured fee per byte will be used
		feePerByte = estimated
		if !feePerByte.IsPositive() {
			feePerByte = configFeePerByte
		}
	}

	feeMax := configFeePerByte
	if req.FeeMax.Valid {
		feeMax = req.FeeMax.Decimal
	}

	if feePerByte.GreaterThan(feeMax) {
		return fmt.Errorf("%w: estimated fee per byte is exceeded: %s > %s", rpccode.GetErrorByCode(rpccode.RPCCodeMaxFeeExceeded), feePerByte, feeMax)
	}

	return nil
}
