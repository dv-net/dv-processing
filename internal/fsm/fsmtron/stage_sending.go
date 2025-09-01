package fsmtron

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

// sending
func (s *FSM) sending(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	fromAddress := s.transfer.GetFromAddress()
	toAddress := s.transfer.GetToAddress()

	// by default use amount from transfer request
	amount := s.transfer.Amount.Decimal

	// get balance for from address
	balance, err := s.getBalance(ctx, fromAddress, s.transfer.AssetIdentifier)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	// if whole amount is true, get wallet balance and use it as amount
	if s.transfer.WholeAmount { //nolint:nestif
		amount = balance

		if s.transfer.AssetIdentifier == tron.TrxAssetIdentifier {
			fee, err := s.bs.Tron().EstimateActivationFee(ctx, fromAddress, toAddress)
			if err != nil {
				return fmt.Errorf("sub activation fee from amount: %w", err)
			}

			amount = amount.Sub(fee.Trx)

			// if transfer is from hot wallet and kind is burn trx, check if we have enough bandwidth
			// if not, use trx for transfer
			if s.transfer.WalletFromType == constants.WalletTypeHot &&
				constants.TronTransferKind(s.transfer.Kind.String) == constants.TronTransferKindBurnTRX {
				resourcesData, err := s.tron.TotalAvailableResources(fromAddress)
				if err != nil {
					return fmt.Errorf("get total available resources: %w", err)
				}

				estimate, err := s.tron.EstimateTransferResources(ctx, fromAddress, toAddress, tron.TrxAssetIdentifier, amount, 6)
				if err != nil {
					return fmt.Errorf("estimate resources: %w", err)
				}

				// if not enough bandwidth, use trx
				if resourcesData.Bandwidth.LessThan(estimate.Bandwidth) {
					amount = amount.Sub(estimate.Trx)
				}
			}
		}
	}

	// if amount is not set, use balance
	if amount.GreaterThan(balance) {
		return fmt.Errorf("transfer amount is greater than balance: %s > %s", amount.String(), balance.String())
	}

	if !amount.IsPositive() {
		return fmt.Errorf("transfer amount is less than or equal to zero: %s", amount.String())
	}

	// get sequence for wallet
	sequence, err := s.bs.Wallets().GetSequenceByWalletType(ctx, s.transfer.WalletFromType, s.transfer.OwnerID, wconstants.BlockchainTypeTron, fromAddress)
	if err != nil {
		return fmt.Errorf("get sequence by wallet type: %w", err)
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	var newTx *api.TransactionExtention
	if s.transfer.AssetIdentifier == tron.TrxAssetIdentifier {
		newTx, err = s.sendTRX(ctx, wcreds, toAddress, amount)
		if err != nil {
			return fmt.Errorf("send trx: %w", err)
		}
	} else {
		assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
		if err != nil {
			return fmt.Errorf("get asset info: %w", err)
		}

		newTx, err = s.sendTrc20(ctx, wcreds, s.transfer.AssetIdentifier, toAddress, amount, assetDecimals, 30_000_000)
		if err != nil {
			return fmt.Errorf("send trc20: %w", err)
		}
	}

	// update transfer and set tx hash
	s.transfer, err = s.bs.Transfers().SetTxHash(ctx, s.transfer.ID, common.Bytes2Hex(newTx.Txid))
	if err != nil {
		return fmt.Errorf("set tx hash: %w", err)
	}

	return nil
}
