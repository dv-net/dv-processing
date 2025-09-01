package transfers

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	"github.com/shopspring/decimal"
)

// processEVM handle transfer request for evm blockchains
func (s *Service) processEVM(ctx context.Context, req *CreateTransferRequest) error {
	switch req.Blockchain {
	case wconstants.BlockchainTypeEthereum:
		if !s.config.Blockchain.Ethereum.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeBinanceSmartChain:
		if !s.config.Blockchain.BinanceSmartChain.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypePolygon:
		if !s.config.Blockchain.Polygon.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeArbitrum:
		if !s.config.Blockchain.Arbitrum.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeOptimism:
		if !s.config.Blockchain.Optimism.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeLinea:
		if !s.config.Blockchain.Linea.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	default:
		return fmt.Errorf("unsupported blockchain type: %s", req.Blockchain.String())
	}

	// check wallet balance
	balance, err := s.eproxySvc.AddressBalance(ctx, req.FromAddresses[0], req.AssetIdentifier, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get wallet [%s] balance: %w", req.FromAddresses[0], err)
	}

	if !req.WholeAmount && balance.LessThanOrEqual(req.Amount.Decimal) {
		return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), req.Amount.Decimal, balance)
	}

	if req.WholeAmount && !balance.IsPositive() {
		return fmt.Errorf("%w for transfer with whole amount, available: 0", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance))
	}

	// check active transfers with the same from address
	transfersInProgress, err := s.store.Transfers().Find(ctx, FindParams{
		StatusesNotIn: []string{
			constants.TransferStatusCompleted.String(),
			constants.TransferStatusFailed.String(),
		},
		FromAddress: &req.FromAddresses[0],
		Blockchain:  &req.Blockchain,
	})
	if err != nil {
		return fmt.Errorf("find transfers: %w", err)
	}

	if len(transfersInProgress) > 0 {
		return rpccode.GetErrorByCode(rpccode.RPCCodeAddressIsTaken)
	}

	assetDecimals, err := s.eproxySvc.AssetDecimals(ctx, req.Blockchain, req.AssetIdentifier)
	if err != nil {
		return fmt.Errorf("get asset decimals: %w", err)
	}

	evmInstance, err := s.blockchains.GetEVMByBlockchain(req.Blockchain)
	if err != nil {
		return fmt.Errorf("get evm instance: %w", err)
	}

	evmConfig, err := s.config.Blockchain.GetEVMByBlockchainType(req.Blockchain)
	if err != nil {
		return fmt.Errorf("get evm config: %w", err)
	}

	maxFeePerGas := decimal.NewFromFloat(evmConfig.GetMaxGasFee())

	// check fee
	if req.walletFromType == constants.WalletTypeHot && //nolint:nestif
		(req.FeeMax.Valid || maxFeePerGas.IsPositive()) {
		amount := req.Amount.Decimal
		if req.WholeAmount {
			amount = balance
		}

		estimateResult, err := evmInstance.EstimateTransfer(ctx, req.FromAddresses[0], req.ToAddresses[0], req.AssetIdentifier, amount, assetDecimals)
		if err != nil {
			return fmt.Errorf("estimate transfer: %w", err)
		}

		var feeMax decimal.Decimal
		if req.FeeMax.Valid {
			feeMax = req.FeeMax.Decimal
		} else {
			feeMax = maxFeePerGas
		}

		totalGasPriceGwei := evm.NewUnit(estimateResult.TotalGasPrice, evm.EtherUnitWei).Value(evm.EtherUnitGWei).Decimal()

		if totalGasPriceGwei.GreaterThan(feeMax) {
			return fmt.Errorf("%w: estimated gas price is exceeded: %s > %s", rpccode.GetErrorByCode(rpccode.RPCCodeMaxFeeExceeded), totalGasPriceGwei, feeMax)
		}
	}

	// check balances
	if req.walletFromType == constants.WalletTypeHot && req.AssetIdentifier != req.Blockchain.GetAssetIdentifier() { //nolint:nestif
		// get processing wallet by owner id
		processingWallet, err := s.walletsSvc.Processing().GetByBlockchain(ctx, req.OwnerID, req.Blockchain)
		if err != nil {
			return fmt.Errorf("get processing wallet: %w", err)
		}

		// get processing wallet balance
		processingWalletBaseAssetBalance, err := s.eproxySvc.AddressBalance(ctx, processingWallet.Address, req.Blockchain.GetAssetIdentifier(), req.Blockchain)
		if err != nil {
			return fmt.Errorf("get processing wallet balance: %w", err)
		}

		hotWalletBaseAssetBalance, err := s.eproxySvc.AddressBalance(ctx, req.FromAddresses[0], req.Blockchain.GetAssetIdentifier(), req.Blockchain)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}

		transferAmount := req.Amount.Decimal
		if req.WholeAmount {
			transferAmount = balance
		}

		// estimateResult transfer resources
		estimateResult, err := evmInstance.EstimateTransfer(ctx, req.FromAddresses[0], req.ToAddresses[0], req.AssetIdentifier, transferAmount, assetDecimals)
		if err != nil {
			return fmt.Errorf("estimate transfer resources: %w", err)
		}

		// calculate total fee in base asset for transfer with coefficient
		transferFeeBaseAssetAmount := evm.NewUnit(estimateResult.TotalFeeAmount.Mul(decimal.NewFromFloat(evm.TransferFeeCoeff)), evm.EtherUnitWei).Value(evm.EtherUnitEther).Decimal()
		req.stateData = map[string]any{
			"handler_estimated_resources": estimateResult,
			"need_fee_amount":             transferFeeBaseAssetAmount,
		}

		// if hot wallet balance is less than fee amount, check processing wallet
		if hotWalletBaseAssetBalance.LessThan(transferFeeBaseAssetAmount) {
			needFeeAmount := transferFeeBaseAssetAmount.Sub(hotWalletBaseAssetBalance)

			req.stateData["need_fee_amount"] = needFeeAmount

			// if processing wallet balance is less than fee amount
			if processingWalletBaseAssetBalance.LessThan(needFeeAmount) {
				return fmt.Errorf(
					"%w %s on processing wallet: fee %s / balance %s",
					rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance),
					req.Blockchain.GetAssetIdentifier(),
					needFeeAmount.String(),
					processingWalletBaseAssetBalance.String(),
				)
			}

			// estimateResult transfer resources
			estimateSendBaseAssetResult, err := evmInstance.EstimateTransfer(ctx, processingWallet.Address, req.FromAddresses[0], req.Blockchain.GetAssetIdentifier(), needFeeAmount, evm.EVMAssetDecimals)
			if err != nil {
				return fmt.Errorf("estimate %s transfer resources: %w", req.Blockchain.GetAssetIdentifier(), err)
			}

			currentTransferFeeAmount := evm.NewUnit(estimateSendBaseAssetResult.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).Decimal()

			// if processing wallet balance is less than fee amount
			if processingWalletBaseAssetBalance.Sub(currentTransferFeeAmount).LessThan(needFeeAmount) {
				return fmt.Errorf(
					"%w %s on processing wallet: fee %s / transfer cost %s / balance %s",
					rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance),
					req.Blockchain.GetAssetIdentifier(),
					needFeeAmount.String(),
					currentTransferFeeAmount.String(),
					processingWalletBaseAssetBalance.String(),
				)
			}
		}
	}

	return nil
}
