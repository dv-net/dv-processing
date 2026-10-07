package fsmltc

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	"github.com/shopspring/decimal"
)

type UTXO struct {
	TxHash   string
	Sequence int32
	// Amount in satoshis
	Amount   decimal.Decimal
	PkScript string
}

// getAddressUTXO
func (s *FSM) getAddressUTXO(ctx context.Context, fromAddress string) ([]UTXO, error) {
	utxosData, err := s.bs.EProxy().GetUTXO(ctx, wconstants.BlockchainTypeLitecoin, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get utxo: %w", err)
	}

	inputs := make(map[string]UTXO, 0)
	for _, item := range utxosData {
		utxoAmount, err := decimal.NewFromString(item.Amount)
		if err != nil {
			return nil, fmt.Errorf("convert amount %s: %w", item.Amount, err)
		}

		utxoAmount = utxoAmount.Mul(assetDecimals)

		if (s.minUTXOAmount.IsPositive() && utxoAmount.LessThan(s.minUTXOAmount)) || utxoAmount.IsZero() {
			continue
		}

		inputs[item.TxHash] = UTXO{
			TxHash:   item.TxHash,
			Sequence: item.Sequence,
			Amount:   utxoAmount,
			PkScript: item.PkScript,
		}
	}

	utxos := make([]UTXO, 0, len(inputs))
	for _, input := range inputs {
		utxos = append(utxos, input)
	}

	return utxos, nil
}

// getAddressesUTXO
func (s *FSM) processAddressesUTXOs(ctx context.Context, owner *models.Owner, newTx *ltc.TxBuilder, addresses []string) (decimal.Decimal, error) {
	var totalUTXOAmount decimal.Decimal
	var err error

	mnemonic := owner.Mnemonic
	if s.config.IsEnabledSeedEncryption() {
		mnemonic, err = encryption.Decrypt(mnemonic, owner.ID.String())
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	// get UTXOs for all addresses
	for _, address := range addresses {
		// get utxo total amount and inputs
		utxos, err := s.getAddressUTXO(ctx, address)
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("prepare transfer: %w", err)
		}

		// get sequence for wallet
		sequence, err := s.bs.Wallets().GetSequenceByWalletType(ctx, s.transfer.WalletFromType, s.transfer.OwnerID, wconstants.BlockchainTypeLitecoin, address)
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("get sequence by wallet type: %w", err)
		}

		addrType, err := s.ltc.WalletSDK.DecodeAddressType(address)
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("decode address type: %w", err)
		}

		addrData, err := s.ltc.WalletSDK.GenerateAddress(addrType, mnemonic, owner.PassPhrase.String, uint32(sequence)) //nolint:gosec
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("get private key for address %s: %w", address, err)
		}

		for _, input := range utxos {
			txInput := ltc.TxInput{
				PrivateKey: addrData.PrivateKey,
				PkScript:   input.PkScript,
				Hash:       input.TxHash,
				Sequence:   uint32(input.Sequence), //nolint:gosec
				Amount:     input.Amount.IntPart(),
			}

			if err := newTx.AddInput(txInput); err != nil {
				return totalUTXOAmount, fmt.Errorf("add transaction input: hash %s, sequence %d: %w", input.TxHash, sequence, err)
			}

			totalUTXOAmount = totalUTXOAmount.Add(input.Amount)
		}
	}

	return totalUTXOAmount, nil
}

// sendFailureEvent
func (s *FSM) sendFailureEvent(ctx context.Context, w *workflow.Workflow, err error, repoOpts ...repos.Option) error {
	params, err := s.bs.Webhooks().EventTransferStatusCreateParams(ctx, webhooks.EventTransferStatusCreateParamsData{
		TransferID:   s.transfer.ID,
		OwnerID:      s.transfer.OwnerID,
		Status:       constants.TransferStatusFailed,
		ErrorMessage: err.Error(),
	})
	if err != nil {
		return fmt.Errorf("get event transfer status create params: %w", err)
	}

	w.State.SetFailed(true).SetError(err)
	w.SetSkipError(true)

	if err := s.bs.Transfers().SetWorkflowSnapshot(ctx, s.transfer.ID, w.GetSnapshot(), repoOpts...); err != nil {
		return fmt.Errorf("set workflow snapshot: %w", err)
	}

	if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{params}, repoOpts...); err != nil {
		return fmt.Errorf("create failed event: %w", err)
	}

	if err := s.bs.Transfers().SetStatus(ctx, s.transfer.ID, constants.TransferStatusFailed, repoOpts...); err != nil {
		return fmt.Errorf("set transfer status %s: %w", constants.TransferStatusFailed, err)
	}

	return nil
}

// setTransferStatus sets the transfer status.
func (s *FSM) setTransferStatus(ctx context.Context, status constants.TransferStatus) error {
	var stepName string
	if s.wf.CurrentStep() != nil {
		stepName = s.wf.CurrentStep().Name
	}

	params, err := s.bs.Webhooks().EventTransferStatusCreateParams(ctx, webhooks.EventTransferStatusCreateParamsData{
		TransferID: s.transfer.ID,
		OwnerID:    s.transfer.OwnerID,
		Step:       stepName,
		Status:     status,
	})
	if err != nil {
		return fmt.Errorf("get event transfer status create params: %w", err)
	}

	if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{params}); err != nil {
		return fmt.Errorf("create failed event: %w", err)
	}

	if err := s.bs.Transfers().SetStatus(ctx, s.transfer.ID, status); err != nil {
		return fmt.Errorf("set transfer status %s: %w", status, err)
	}

	return nil
}

// addOutputsAndFee adds the recipient output and, for a transfer with amount, the change output
// back to the sender address. The fee is subtracted from the change output, or from the recipient
// output for a whole amount transfer.
func (s *FSM) addOutputsAndFee(
	newTx *ltc.TxBuilder,
	toAddress string,
	transferAmount, amountRemaining, feePerByte decimal.Decimal,
) (ltc.CalculateTxSizeData, error) {
	var txSizeData ltc.CalculateTxSizeData
	var err error

	if !s.transfer.WholeAmount && transferAmount.LessThan(dustThreshold) {
		return txSizeData, fmt.Errorf("transfer amount %s is less than dust threshold %s", transferAmount.String(), dustThreshold.String())
	}

	// set output
	if err := newTx.AddOutput(toAddress, transferAmount); err != nil {
		return txSizeData, fmt.Errorf("add transaction output for address %s: %w", toAddress, err)
	}

	// For a transfer with amount the remaining amount is sent back to the sender address
	// and the fee is paid from it, so the recipient receives exactly the requested amount.
	feeOutputIdx := 0
	if !s.transfer.WholeAmount {
		if err := newTx.AddOutput(s.transfer.FromAddresses[0], amountRemaining); err != nil {
			return txSizeData, fmt.Errorf("add change output for address %s: %w", s.transfer.FromAddresses[0], err)
		}

		feeOutputIdx = 1
	}

	// emulate transaction and calculate fee
	txSizeData, err = newTx.EmulateTxSize(feePerByte)
	if err != nil {
		return txSizeData, fmt.Errorf("emulate transaction size: %w", err)
	}

	// Change after the fee would be a dust output rejected by the network,
	// so drop it and leave the whole remaining amount as the fee.
	if !s.transfer.WholeAmount && amountRemaining.Sub(txSizeData.TotalFee).LessThan(dustThreshold) {
		newTx.MsgTx().TxOut = newTx.MsgTx().TxOut[:1]
		feeOutputIdx = -1

		txSizeData, err = newTx.EmulateTxSize(feePerByte)
		if err != nil {
			return txSizeData, fmt.Errorf("emulate transaction size without change: %w", err)
		}

		if amountRemaining.LessThan(txSizeData.TotalFee) {
			return txSizeData, fmt.Errorf(
				"insufficient funds for fee: remaining %s, fee %s",
				amountRemaining.String(), txSizeData.TotalFee.String(),
			)
		}

		txSizeData.TotalFee = amountRemaining
	}

	// set fee to the original transaction
	if feeOutputIdx >= 0 {
		newTx.MsgTx().TxOut[feeOutputIdx].Value -= txSizeData.TotalFee.IntPart()
	}

	return txSizeData, nil
}

// getFeePerByte returns the fee per byte for the transfer: from the request if it is set,
// otherwise the current network rate estimated by the node. If the node has no estimate yet,
// the configured fee per byte is used.
//
// The fee per byte must not exceed the max fee from the request or,
// for transfers from hot wallets, the configured fee per byte.
func (s *FSM) getFeePerByte() (decimal.Decimal, error) {
	var feePerByte decimal.Decimal
	switch {
	case s.transfer.Fee.Valid && s.transfer.Fee.Decimal.IsPositive():
		feePerByte = s.transfer.Fee.Decimal
	case s.config.Blockchain.Litecoin.Network == "testnet":
		feePerByte = decimal.NewFromInt(5)
	default:
		estimated, err := s.ltc.EstimateFeePerByte()
		if err != nil {
			return decimal.Zero, fmt.Errorf("estimate fee per byte: %w", err)
		}

		feePerByte = estimated
		if !feePerByte.IsPositive() {
			feePerByte = s.feePerByte
		}
	}

	if !feePerByte.IsPositive() {
		return decimal.Zero, fmt.Errorf("fee per byte is not estimated by the node and not set in the config")
	}

	var maxFeePerByte decimal.Decimal
	if s.transfer.FeeMax.Valid {
		maxFeePerByte = s.transfer.FeeMax.Decimal
	} else if s.transfer.WalletFromType == constants.WalletTypeHot {
		maxFeePerByte = s.feePerByte
	}

	if maxFeePerByte.IsPositive() && feePerByte.GreaterThan(maxFeePerByte) {
		return decimal.Zero, fmt.Errorf(
			"%w: fee per byte is exceeded: %s > %s",
			rpccode.GetErrorByCode(rpccode.RPCCodeMaxFeeExceeded), feePerByte.String(), maxFeePerByte.String(),
		)
	}

	return feePerByte, nil
}
