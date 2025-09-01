package fsmbtc

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/mx/logger"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
)

type FSM struct {
	logger   logger.Logger
	config   *config.Config
	wf       *workflow.Workflow
	transfer *models.Transfer
	st       store.IStore

	btc *btc.BTC

	feePerByte    decimal.Decimal
	minUTXOAmount decimal.Decimal

	// services
	bs baseservices.IBaseServices
}

func NewFSM(
	l logger.Logger,
	conf *config.Config,
	st store.IStore,
	bs baseservices.IBaseServices,
	transfer *models.Transfer,
) (*FSM, error) {
	fsm := &FSM{
		logger:   l,
		config:   conf,
		st:       st,
		bs:       bs,
		btc:      bs.BTC(),
		transfer: transfer,
	}

	if conf.Blockchain.Bitcoin.Attributes.FeePerByte > 0 {
		fsm.feePerByte = decimal.NewFromInt(conf.Blockchain.Bitcoin.Attributes.FeePerByte)
	}

	if conf.Blockchain.Bitcoin.Attributes.MinUTXOAmount > 0 {
		fsm.minUTXOAmount = decimal.NewFromInt(conf.Blockchain.Bitcoin.Attributes.MinUTXOAmount)
	}

	// create a workflow
	fsm.wf = workflow.New(
		workflow.WithName("BTC FSM"),
		workflow.WithLogger(l),
		workflow.WithDebug(true),
		workflow.WithBeforeAllStepsFn(func(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
				return fmt.Errorf("set workflow snapshot: %w", err)
			}

			return nil
		}),
		workflow.WithAfterAllStepsFn(func(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
				return fmt.Errorf("set workflow snapshot: %w", err)
			}

			return nil
		}),
		workflow.WithAfterFn(func(ctx context.Context, _ *workflow.Workflow) error {
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
				return fmt.Errorf("set workflow snapshot: %w", err)
			}

			return nil
		}),
	).SetOnFailureFn(func(ctx context.Context, w *workflow.Workflow, err error) error {
		if err != nil && w.CurrentStage() != nil && w.CurrentStage().Name != stageAfterSending {
			return fsm.sendFailureEvent(ctx, w, err)
		}

		return nil
	})

	// set stages and steps for the workflow
	fsm.wf.SetStages([]*workflow.Stage{
		// before sending
		{
			Name: stageBeforeSending,
			Steps: []*workflow.Step{
				{
					Name: stepValidateRequest,
					Func: fsm.validateRequest,
				},
			},
		},
		// sending
		{
			Name: stageSending,
			Steps: []*workflow.Step{
				{
					Name: stepSending,
					Func: fsm.sendTransfer,
				},
			},
		},
		// after sending
		{
			Name: stageAfterSending,
			Steps: []*workflow.Step{
				{
					Name: stepWaitingInMempool,
					Func: fsm.waitingInMempool,
				},
				{
					Name: stepWaitingForTheFirstConfirmation,
					Func: fsm.waitingForTheFirstConfirmation,
				},
				{
					Name: stepWaitingConfirmations,
					Func: fsm.waitingConfirmations,
				},
				{
					Name: stepSendSuccessEvent,
					Func: fsm.sendSuccessEvent,
				},
			},
		},
	})

	// set snapshot for the workflow
	if err := fsm.wf.SetSnapshot(transfer.WorkflowSnapshot); err != nil {
		return nil, fmt.Errorf("set workflow snapshot for transfer %s: %w", transfer.ID.String(), err)
	}

	return fsm, nil
}

// validateRequest
func (s *FSM) validateRequest(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if len(s.transfer.FromAddresses) < 1 {
		return fmt.Errorf("at least one from address is required")
	}

	if len(s.transfer.ToAddresses) != 1 {
		return fmt.Errorf("required one to address")
	}

	if !s.transfer.WholeAmount {
		return fmt.Errorf("only whole amount is supported for bitcoin transfers")
	}

	// check cold or processing wallet
	if s.transfer.WalletFromType == constants.WalletTypeHot {
		checkResult, err := s.bs.Wallets().CheckWallet(ctx, wconstants.BlockchainTypeBitcoin, s.transfer.GetToAddress())
		if err != nil {
			return fmt.Errorf("check wallet: %w", err)
		}

		if !slices.Contains([]constants.WalletType{constants.WalletTypeCold, constants.WalletTypeProcessing}, checkResult.WalletType) {
			return fmt.Errorf("invalid wallet type %s", checkResult.WalletType)
		}

		if checkResult.OwnerID != s.transfer.OwnerID {
			return fmt.Errorf("invalid wallet owner %s", checkResult.OwnerID)
		}
	}

	return s.setTransferStatus(ctx, constants.TransferStatusProcessing)
}

// send transfer
func (s *FSM) sendTransfer(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if len(s.transfer.FromAddresses) < 1 {
		return fmt.Errorf("at least one from address is required")
	}

	if len(s.transfer.ToAddresses) != 1 {
		return fmt.Errorf("required one to address")
	}

	// configure fee per byte
	feePerByte := s.feePerByte
	if s.config.Blockchain.Bitcoin.Network == "testnet" {
		feePerByte = decimal.NewFromInt(5)
	}

	// use fee from request if it is set
	if s.transfer.Fee.Valid && s.transfer.Fee.Decimal.GreaterThan(decimal.Zero) {
		feePerByte = s.transfer.Fee.Decimal
	}

	// check max fee if it is set
	if s.transfer.FeeMax.Valid &&
		s.transfer.FeeMax.Decimal.IsPositive() &&
		feePerByte.GreaterThan(s.transfer.FeeMax.Decimal) {
		return fmt.Errorf("fee per byte %s is greater than max fee %s", feePerByte.String(), s.transfer.FeeMax.Decimal.String())
	}

	toAddress := s.transfer.GetToAddress()

	// get owner
	owner, err := s.bs.Owners().GetByID(ctx, s.transfer.OwnerID)
	if err != nil {
		return fmt.Errorf("get owner: %w", err)
	}

	// validate pass phrase
	// if !owner.PassPhrase.Valid || owner.PassPhrase.String == "" {
	// 	return fmt.Errorf("empty passphrase")
	// }

	// define a new transaction builder
	newTx := btc.NewTxBuilder(s.btc.WalletSDK.ChainParams())

	totalUTXOAmount, err := s.processAddressesUTXOs(ctx, owner, newTx, s.transfer.FromAddresses)
	if err != nil {
		return fmt.Errorf("get addresses utxo: %w", err)
	}

	if !totalUTXOAmount.IsPositive() {
		return fmt.Errorf("total utxo amount is less than or equal to zero: %s", totalUTXOAmount.String())
	}

	// If we withdraw all funds from addresses, then the return amount should be zero.
	// If we withdraw a specific amount, then we indicate this amount in the output, the rest, taking into account the commission,
	// is returned back to the first input. Otherwise, this entire amount will burn and go to the miners.
	var transferAmount, amountRemaining decimal.Decimal
	if s.transfer.WholeAmount {
		// transferAmount = totalUTXOAmount.Sub(fee)
		transferAmount = totalUTXOAmount
	} else {
		// amountRemaining = totalUTXOAmount.Sub(transferAmount.Sub(fee))
		transferAmount = s.transfer.Amount.Decimal.Mul(assetDecimals)
		amountRemaining = totalUTXOAmount.Sub(transferAmount)
	}

	s.logger.Infow("transfer data",
		"amount", totalUTXOAmount.String(),
		"fee_per_byte", feePerByte.String(),
		"min_utxo_amount", s.minUTXOAmount.String(),
		"transfer_amount", transferAmount.String(),
		"amount_remaining", amountRemaining.String(),
		"whole_amount", s.transfer.WholeAmount,
		"requested_amount", s.transfer.Amount.Decimal.String(),
		"requested_fee", s.transfer.Fee.Decimal.String(),
		"requested_fee_max", s.transfer.FeeMax.Decimal.String(),
	)

	if !transferAmount.IsPositive() {
		return fmt.Errorf(
			"transferAmount is not positive %s",
			transferAmount.Div(assetDecimals).String(),
		)
	}

	if amountRemaining.IsNegative() {
		return fmt.Errorf("amount remaining is negative: %s", amountRemaining.String())
	}

	// set output
	if err := newTx.AddOutput(toAddress, transferAmount); err != nil {
		return fmt.Errorf("add transaction output for address %s: %w", toAddress, err)
	}

	// send remaining amount back
	// TODO: send the amount back to the desired wallet
	if amountRemaining.IsPositive() {
		if err := newTx.AddOutput(s.transfer.FromAddresses[0], amountRemaining); err != nil {
			return fmt.Errorf("add transaction output for address %s: %w", s.transfer.FromAddresses[0], err)
		}
	}

	// emulate transaction and calculate fee
	txSizeData, err := newTx.EmulateTxSize(feePerByte)
	if err != nil {
		return fmt.Errorf("emulate transaction size: %w", err)
	}

	// set fee to the original transaction
	newTx.MsgTx().TxOut[0].Value -= txSizeData.TotalFee.IntPart()

	// sign original transaction
	if err := newTx.SignTx(); err != nil {
		return fmt.Errorf("sign transaction: %w", err)
	}

	s.logger.Infow("transaction data last",
		"tx_full_size", txSizeData.TxFullSize.String(),
		"tx_stripped_size", txSizeData.TxStrippedSize.String(),
		"weight", txSizeData.Weight.String(),
		"v_size", txSizeData.VSize.String(),
		"total_fee", txSizeData.TotalFee.String(),
		"result_value", newTx.MsgTx().TxOut[0].Value,
	)

	if newTx.MsgTx().TxOut[0].Value < 0 {
		return fmt.Errorf("result value is less than zero: %d", newTx.MsgTx().TxOut[0].Value)
	}

	// update transfer and set tx hash
	s.transfer, err = s.bs.Transfers().SetTxHash(ctx, s.transfer.ID, newTx.MsgTx().TxHash().String())
	if err != nil {
		return fmt.Errorf("set tx hash: %w", err)
	}

	stateData := map[string]any{
		"from":              s.transfer.FromAddresses,
		"to":                toAddress,
		"tx_full_size":      txSizeData.TxFullSize.String(),
		"tx_stripped_size":  txSizeData.TxStrippedSize.String(),
		"weight":            txSizeData.Weight.String(),
		"v_size":            txSizeData.VSize.String(),
		"total_fee":         txSizeData.TotalFee.String(),
		"fee_per_byte":      feePerByte.String(),
		"min_utxo_amount":   s.minUTXOAmount.String(),
		"transfer_amount":   transferAmount.String(),
		"amount_remaining":  amountRemaining.String(),
		"result_value":      newTx.MsgTx().TxOut[0].Value,
		"whole_amount":      s.transfer.WholeAmount,
		"requested_amount":  s.transfer.Amount.Decimal.Mul(assetDecimals).String(),
		"requested_fee":     s.transfer.Fee.Decimal.String(),
		"requested_fee_max": s.transfer.FeeMax.Decimal.String(),
	}

	// set state data
	if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, stateData); err != nil {
		return fmt.Errorf("set state data: %w", err)
	}

	// send transaction
	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := s.btc.Node().SendRawTransaction(newTx.MsgTx(), false)
		if err != nil {
			err = fmt.Errorf("failed to send transaction [%s]: %w", newTx.MsgTx().TxHash().String(), err)
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("send transaction timeout: %w", ctx.Err())
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	return nil
}

// waitingInMempool
func (s *FSM) waitingInMempool(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	txs, err := s.bs.EProxy().FindTransactions(ctx, wconstants.BlockchainTypeBitcoin, eproxy.FindTransactionsParams{
		Hash: &s.transfer.TxHash.String,
	})
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return workflow.NoConsoleError(river.JobSnooze(time.Second * 3))
	}

	if err := s.setTransferStatus(ctx, constants.TransferStatusInMempool); err != nil {
		return err
	}

	return nil
}

// waitingForTheFirstConfirmation
func (s *FSM) waitingForTheFirstConfirmation(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, wconstants.BlockchainTypeBitcoin, s.transfer.TxHash.String)
	if err != nil {
		if strings.Contains(err.Error(), "data not found") {
			return workflow.NoConsoleError(river.JobSnooze(time.Second * 5))
		}
		return fmt.Errorf("find transaction: %w", err)
	}

	if tx.Confirmations == 0 {
		delay := constants.ConfirmationsTimeout(wconstants.BlockchainTypeBitcoin, constants.GetMinConfirmations(wconstants.BlockchainTypeBitcoin)-1)
		return workflow.NoConsoleError(river.JobSnooze(delay))
	}

	if err := s.setTransferStatus(ctx, constants.TransferStatusUnconfirmed); err != nil {
		return err
	}

	return nil
}

// waitingConfirmations
func (s *FSM) waitingConfirmations(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, wconstants.BlockchainTypeBitcoin, s.transfer.TxHash.String)
	if err != nil {
		return fmt.Errorf("find transaction: %w", err)
	}

	delay := constants.ConfirmationsTimeout(wconstants.BlockchainTypeBitcoin, tx.Confirmations)
	minConfirmationsCount := constants.GetMinConfirmations(wconstants.BlockchainTypeBitcoin)
	if tx.Confirmations < minConfirmationsCount {
		return workflow.NoConsoleError(river.JobSnooze(delay))
	}

	return nil
}

// sendSuccessEvent
func (s *FSM) sendSuccessEvent(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if err := s.setTransferStatus(ctx, constants.TransferStatusCompleted); err != nil {
		return err
	}
	return nil
}

// Run executes the workflow.
func (s *FSM) Run(ctx context.Context) error {
	return s.wf.Run(
		constants.WithClientContext(ctx, s.transfer.ClientID),
	)
}
