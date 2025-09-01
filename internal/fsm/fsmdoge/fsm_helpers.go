package fsmdoge

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
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
	utxosData, err := s.bs.EProxy().GetUTXO(ctx, wconstants.BlockchainTypeDogecoin, fromAddress)
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
func (s *FSM) processAddressesUTXOs(ctx context.Context, owner *models.Owner, newTx *doge.TxBuilder, addresses []string) (decimal.Decimal, error) {
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
		sequence, err := s.bs.Wallets().GetSequenceByWalletType(ctx, s.transfer.WalletFromType, s.transfer.OwnerID, wconstants.BlockchainTypeDogecoin, address)
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("get sequence by wallet type: %w", err)
		}

		addrData, err := s.doge.WalletSDK.GenerateAddress(mnemonic, owner.PassPhrase.String, uint32(sequence)) //nolint:gosec
		if err != nil {
			return totalUTXOAmount, fmt.Errorf("get private key for address %s: %w", address, err)
		}

		for _, input := range utxos {
			txInput := doge.TxInput{
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
