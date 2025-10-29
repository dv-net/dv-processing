package fsmtron

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	trxv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"

	commonv1 "github.com/dv-net/dv-proto/gen/go/manager/common/v1"
	orderv1 "github.com/dv-net/dv-proto/gen/go/manager/order/v1"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfer_transactions"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/errutils"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

func getDelegateResourceKey(resourceType core.ResourceCode) string {
	return strings.ToLower(fmt.Sprintf("delegate_%s", resourceType.String()))
}

func getReclaimResourceKey(resourceType core.ResourceCode) string {
	return strings.ToLower(fmt.Sprintf("reclaim_%s", resourceType.String()))
}

func getTransferKind(kind string) (constants.TronTransferKind, error) {
	if kind == "" {
		return "", fmt.Errorf("transfer kind is empty")
	}

	transferKind := constants.TronTransferKind(kind)
	if !transferKind.Valid() {
		return "", fmt.Errorf("invalid transfer kind: %s", kind)
	}

	return transferKind, nil
}

// setTransferStatus sets the transfer status.
func (s *FSM) setTransferStatus(ctx context.Context, status constants.TransferStatus, repoOpts ...repos.Option) error {
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

	if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{params}, repoOpts...); err != nil {
		return fmt.Errorf("create failed event: %w", err)
	}

	if err := s.bs.Transfers().SetStatus(ctx, s.transfer.ID, status, repoOpts...); err != nil {
		return fmt.Errorf("set transfer status %s: %w", status, err)
	}

	return nil
}

// createTransferStepWh create wh with changed transfer step
func (s *FSM) createTransferStepWh(ctx context.Context, step *workflow.Step) error {
	params, err := s.bs.Webhooks().EventTransferStatusCreateParams(ctx, webhooks.EventTransferStatusCreateParamsData{
		TransferID: s.transfer.ID,
		OwnerID:    s.transfer.OwnerID,
		Status:     s.transfer.Status,
		Step:       step.Name,
	})
	if err != nil {
		return fmt.Errorf("get event transfer status create params: %w", err)
	}

	if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{params}); err != nil {
		return fmt.Errorf("create failed event: %w", err)
	}

	return nil
}

func (s *FSM) checkTransactionConfirmations(ctx context.Context, transferTx *models.TransferTransaction, confirmationsCount uint64) error {
	// get transaction info
	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, wconstants.BlockchainTypeTron, transferTx.TxHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return workflow.NoConsoleError(river.JobSnooze(time.Second))
		}

		return fmt.Errorf("get transaction info: %w", err)
	}

	if tx.GetStatus() != "success" {
		if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusFailed); err != nil {
			return fmt.Errorf("update system transaction expenses: %w", err)
		}

		return newErrorFailedTransfer(fmt.Errorf("transaction status is not success: %s", tx.GetStatus()), s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	}

	// if transaction is not confirmed, snooze the job
	if tx.Confirmations < confirmationsCount {
		if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusUnconfirmed); err != nil {
			return fmt.Errorf("update system transaction expenses: %w", err)
		}

		return workflow.NoConsoleError(river.JobSnooze(
			constants.ConfirmationsTimeoutWithRequired(wconstants.BlockchainTypeTron, confirmationsCount, tx.Confirmations),
		))
	}

	if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusConfirmed); err != nil {
		return fmt.Errorf("update system transaction expenses: %w", err)
	}

	return nil
}

// getBalance returns the balance of the address.
func (s *FSM) getBalance(ctx context.Context, address string, assetIdentifier string) (decimal.Decimal, error) {
	balance, err := s.bs.EProxy().AddressBalance(ctx, address, assetIdentifier, wconstants.BlockchainTypeTron)
	if err != nil {
		return decimal.Zero, err
	}

	return balance, nil
}

func (s *FSM) getAssetDecimals(ctx context.Context, assetIdentifier string) (int64, error) {
	return s.bs.EProxy().AssetDecimals(ctx, wconstants.BlockchainTypeTron, assetIdentifier)
}

type walletCreds struct {
	Address    string
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// getWalletCreds
func (s *FSM) getWalletCreds(ctx context.Context, ownerID uuid.UUID, sequence uint32) (*walletCreds, error) {
	// get owner
	owner, err := s.bs.Owners().GetByID(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("get owner: %w", err)
	}

	mnemonic := owner.Mnemonic
	if s.config.IsEnabledSeedEncryption() {
		mnemonic, err = encryption.Decrypt(mnemonic, owner.ID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	address, priv, public, err := tron.WalletPubKeyHash(mnemonic, owner.PassPhrase.String, sequence)
	if err != nil {
		return nil, err
	}

	return &walletCreds{
		Address:    address,
		PrivateKey: priv,
		PublicKey:  public,
	}, nil
}

func (s *FSM) systemActivation(ctx context.Context, wCreds *walletCreds, toAddress string) (*api.TransactionExtention, error) {
	if toAddress == "" {
		return nil, fmt.Errorf("to address is not valid")
	}

	tx, err := s.tron.Node().CreateAccount(wCreds.Address, toAddress)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	if !tx.Result.Result {
		return nil, fmt.Errorf("create system activation tx error: %s", string(tx.Result.Message))
	}

	if err := s.tron.SignTransaction(tx.GetTransaction(), wCreds.PrivateKey); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, tx, dbTx); err != nil {
			return fmt.Errorf("store system transaction info: %w", err)
		}

		if _, err := s.tron.Node().Broadcast(tx.GetTransaction()); err != nil {
			return fmt.Errorf("broadcast transaction: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	return tx, nil
}

// sendTRX allows to send TRX berween wallets.
//
// Important: Amount in TRX. You can use non-integer values, for example, 0.1 TRX. Amount will be converted to SUN automatically.
func (s *FSM) sendTRX(ctx context.Context, wCreds *walletCreds, toAddress string, amount decimal.Decimal) (*api.TransactionExtention, error) {
	if toAddress == "" {
		return nil, fmt.Errorf("to address is not valid")
	}

	amount = amount.Mul(decimal.NewFromInt(1e6))

	if !amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than 0: %s", amount.String())
	}

	s.logger.Infow(
		"send trx",
		"from", wCreds.Address,
		"to", toAddress,
		"amount", amount.IntPart(),
	)

	tx, err := s.tron.Node().Transfer(wCreds.Address, toAddress, amount.IntPart())
	if err != nil {
		return nil, fmt.Errorf("transfer: %w", err)
	}

	if !tx.Result.Result {
		return nil, fmt.Errorf("create send trx tx error: %s", string(tx.Result.Message))
	}

	if err := s.tron.SignTransaction(tx.GetTransaction(), wCreds.PrivateKey); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	stateData := map[string]any{
		"amount": amount,
	}

	if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, stateData); err != nil {
		return nil, fmt.Errorf("set state data: %w", err)
	}

	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, tx, dbTx); err != nil {
			return fmt.Errorf("store transaction info: %w", err)
		}

		if _, err = s.tron.Node().Broadcast(tx.GetTransaction()); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("broadcast transaction: %w", err)
	}

	return tx, nil
}

func (s *FSM) sendTrc20(ctx context.Context, wCreds *walletCreds, contractAddress string, toAddress string, amount decimal.Decimal, decimals int64, feeLimit int64) (*api.TransactionExtention, error) {
	if contractAddress == tron.TrxAssetIdentifier || contractAddress == "" {
		return nil, fmt.Errorf("contract address is not valid")
	}

	if toAddress == "" {
		return nil, fmt.Errorf("to address is not valid")
	}

	amount = amount.Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(decimals))))

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	if feeLimit <= 0 {
		return nil, fmt.Errorf("fee limit must be greater than 0")
	}

	s.logger.Infow(
		"send trc20",
		"from", wCreds.Address,
		"to", toAddress,
		"contract", contractAddress,
		"amount", amount.String(),
		"fee_limit", feeLimit,
		"decimals", decimals,
	)

	tx, err := s.tron.Node().TRC20Send(wCreds.Address, toAddress, contractAddress, amount.BigInt(), feeLimit)
	if err != nil {
		return nil, fmt.Errorf("cannot make tron transaction: %w", err)
	}

	if !tx.Result.Result {
		return nil, fmt.Errorf("create send trc20 tx error: %s", string(tx.Result.Message))
	}

	if err := s.tron.SignTransaction(tx.GetTransaction(), wCreds.PrivateKey); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	stateData := map[string]any{
		"contract_address": contractAddress,
		"amount":           amount,
		"fee_limit":        feeLimit,
		"decimals":         decimals,
	}

	if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, stateData); err != nil {
		return nil, fmt.Errorf("set state data: %w", err)
	}

	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, tx, dbTx); err != nil {
			return fmt.Errorf("store system transaction info: %w", err)
		}

		if _, err = s.tron.Node().Broadcast(tx.GetTransaction()); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("broadcast transaction: %w", err)
	}

	return tx, nil
}

// delegateResource
func (s *FSM) delegateResource(
	ctx context.Context,
	wCreds *walletCreds,
	toAddress string,
	amount decimal.Decimal,
	resourceType core.ResourceCode,
) (*api.TransactionExtention, *delegateStateData, error) {
	if toAddress == "" {
		return nil, nil, fmt.Errorf("to address is not valid")
	}

	if !amount.IsPositive() {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}

	chainParams, err := s.tron.ChainParams(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get chain params: %w", err)
	}

	accountResources, err := s.tron.Node().GetAccountResource(wCreds.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("get account resources: %w", err)
	}

	var delegateBalance int64
	switch resourceType {
	case core.ResourceCode_ENERGY:
		delegateBalance = s.tron.ConvertEnergyToStackedTRX(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, amount).Ceil().IntPart()
	case core.ResourceCode_BANDWIDTH:
		delegateBalance = s.tron.ConvertBandwidthToStackedTRX(accountResources.TotalNetWeight, accountResources.TotalNetLimit, amount).Ceil().IntPart()
	default:
		return nil, nil, fmt.Errorf("unsupported resource type: %s", resourceType.String())
	}

	s.logger.Infow(
		"delegate resource",
		"from", wCreds.Address,
		"to", toAddress,
		"amount", amount.String(),
		"coeff", tron.ResourceCoefficient.String(),
		"amount_in_trx", delegateBalance,
		"resource", resourceType.String(),
		"request_id", s.transfer.RequestID,
	)

	tx, err := s.tron.Node().DelegateResource(wCreds.Address, toAddress, resourceType, delegateBalance, false, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("delegate resource: %w", err)
	}

	if !tx.Result.Result {
		return nil, nil, fmt.Errorf("create delegate tx error: %s", string(tx.Result.Message))
	}

	if err := s.tron.SignTransaction(tx.GetTransaction(), wCreds.PrivateKey); err != nil {
		return nil, nil, fmt.Errorf("sign transaction: %w", err)
	}

	delegateData := &delegateStateData{
		TxHash:      common.Bytes2Hex(tx.Txid),
		Amount:      amount,
		Coeff:       tron.ResourceCoefficient,
		AmountInTrx: delegateBalance,
		FromAddress: wCreds.Address,
		ToAddress:   toAddress,
	}

	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, tx, dbTx); err != nil {
			return fmt.Errorf("store system transaction info: %w", err)
		}

		if _, err = s.tron.Node().Broadcast(tx.GetTransaction()); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, nil, fmt.Errorf("broadcast transaction: %w", err)
	}

	return tx, delegateData, nil
}

func (s *FSM) reclaimResource(ctx context.Context, wCreds *walletCreds, toAddress string, amount int64, resourceType core.ResourceCode) (*api.TransactionExtention, *reclaimStateData, error) {
	if toAddress == "" {
		return nil, nil, fmt.Errorf("to address is not valid")
	}

	if amount <= 0 {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}

	s.logger.Infow(
		"reclaim resource",
		"from", wCreds.Address,
		"to", toAddress,
		"amount", amount,
		"resource", resourceType.String(),
		"hash", s.transfer.TxHash.String,
	)

	tx, err := s.tron.Node().UnDelegateResource(wCreds.Address, toAddress, resourceType, amount)
	if err != nil {
		return nil, nil, err
	}

	if !tx.Result.Result {
		return nil, nil, fmt.Errorf("create reclaim tx error: %s", string(tx.Result.Message))
	}

	if err := s.tron.SignTransaction(tx.GetTransaction(), wCreds.PrivateKey); err != nil {
		return nil, nil, fmt.Errorf("sign transaction: %w", err)
	}

	reclaimData := &reclaimStateData{
		TxHash:      common.Bytes2Hex(tx.Txid),
		Amount:      amount,
		FromAddress: wCreds.Address,
		ToAddress:   toAddress,
	}

	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, tx, dbTx); err != nil {
			return fmt.Errorf("store system transaction info: %w", err)
		}

		if _, err = s.tron.Node().Broadcast(tx.GetTransaction()); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, nil, fmt.Errorf("broadcast transaction: %w", err)
	}

	return tx, reclaimData, nil
}

func (s *FSM) checkExternalOrderStatus(ctx context.Context, orderID string) error {
	var order *connect.Response[orderv1.GetOrderResponse]
	var err error

	retryClient := retry.New(
		retry.WithMaxAttempts(2),
		retry.WithPolicy(retry.PolicyLinear),
		retry.WithLogger(s.logger),
	)

	err = retryClient.Do(func() error {
		order, err = s.bs.RManager().OrdersClient().GetOrder(ctx, connect.NewRequest(&orderv1.GetOrderRequest{
			OrderId: orderID,
		}))
		if err != nil && !errutils.IsNetworkError(err) {
			return retry.ErrExit
		}
		return err
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return fmt.Errorf("order %s not found", orderID)
		}
		return newErrorFailedTransfer(fmt.Errorf("get order by id: %w", err), s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	}
	switch order.Msg.GetOrder().GetOrderStatus() {
	case commonv1.OrderStatus_ORDER_STATUS_PENDING, commonv1.OrderStatus_ORDER_STATUS_IN_PROGRESS:
		// Order either was not started yet or is in progress, we can snooze the job
		return workflow.NoConsoleError(river.JobSnooze(
			time.Second,
		))
	case commonv1.OrderStatus_ORDER_STATUS_COMPLETED:
		if order.Msg.GetOrder().GetOrderType() == commonv1.OrderType_ORDER_TYPE_TRON_ENERGY_DELEGATION || order.Msg.GetOrder().GetOrderType() == commonv1.OrderType_ORDER_TYPE_TRON_BANDWIDTH_DELEGATION {
			return fmt.Errorf("order %s appears to be completed before transfer", orderID)
		}
		return nil
	case commonv1.OrderStatus_ORDER_STATUS_FULFILLED:
		// Order is fulfilled, we can proceed with the next step
		return nil
	case commonv1.OrderStatus_ORDER_STATUS_FAILED:
		// Order is failed, we need to handle it
		return newErrorFailedTransfer(fmt.Errorf("order id %s failed", orderID), s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	// TODO: handle canceled order
	default:
		return newErrorFailedTransfer(fmt.Errorf("order %s unsupported order status: %v", order.Msg.GetOrder().GetOrderId(), order.Msg.GetOrder().GetOrderStatus()), s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	}
}

// initPendingSystemTransaction extracts and combines transaction expenses for a TRON transaction.
// It creates new expenses (energy, bandwidth, and burnt native token) to the transfer_transactions table.
func (s *FSM) initPendingSystemTransaction(ctx context.Context, txInfo *api.TransactionExtention, dbTx pgx.Tx) error {
	txType, err := s.prepareTransferTransactionTypeByStep()
	if err != nil {
		return fmt.Errorf("prepare transfer transaction type: %w", err)
	}

	if _, err = s.st.TransferTransactions(repos.WithTx(dbTx)).Create(ctx, repo_transfer_transactions.CreateParams{
		TransferID: s.transfer.ID,
		TxHash:     common.Bytes2Hex(txInfo.Txid),
		TxType:     *txType,
		Status:     models.TransferTransactionsStatusPending,
		Step:       s.wf.CurrentStep().Name,
	}); err != nil {
		return fmt.Errorf("create transfer transaction: %w", err)
	}

	return nil
}

// updateSystemTransactionStatus extracts and combines transaction expenses for a TRON transaction.
// It updates (energy, bandwidth, and burnt native token) in the transfer_transactions table.
func (s *FSM) updateSystemTransactionStatus(ctx context.Context, tx *trxv2.Transaction, status models.TransferTransactionsStatus) error {
	// Validate input transaction and TRON data
	if tx == nil || tx.AdditionalData == nil || tx.AdditionalData.Tron == nil {
		return fmt.Errorf("invalid transaction: missing TRON data")
	}

	// Get transaction hash and workflow step
	txHash := tx.GetHash()
	if txHash == "" {
		return fmt.Errorf("invalid transaction: empty hash")
	}

	// Parse TRON-specific expenses
	energyUsage, err := decimal.NewFromString(tx.AdditionalData.Tron.GetEnergyUsage())
	if err != nil {
		return fmt.Errorf("failed to parse energy usage: %w", err)
	}
	bandwidthUsage, err := decimal.NewFromString(tx.AdditionalData.Tron.GetNetUsage())
	if err != nil {
		return fmt.Errorf("failed to parse bandwidth usage: %w", err)
	}
	burntTrxForBandwidth, err := decimal.NewFromString(tx.AdditionalData.Tron.GetNetFee())
	if err != nil {
		return fmt.Errorf("failed to parse burnt TRX for bandwidth: %w", err)
	}
	burntTrxForEnergy, err := decimal.NewFromString(tx.AdditionalData.Tron.GetEnergyFee())
	if err != nil {
		return fmt.Errorf("failed to parse burnt TRX for energy: %w", err)
	}

	nativeTokenAmount := decimal.Zero
	if tx.GetAssetIdentifier() == tron.TrxAssetIdentifier {
		nativeTokenAmount, err = decimal.NewFromString(tx.GetAmount())
		if err != nil {
			return fmt.Errorf("failed to parse amount: %w", err)
		}
	}

	return s.st.TransferTransactions().UpdatePendingTxExpense(ctx, repo_transfer_transactions.UpdatePendingTxExpenseParams{
		BandwidthAmount:   bandwidthUsage,
		EnergyAmount:      energyUsage,
		NativeTokenAmount: nativeTokenAmount,
		NativeTokenFee:    tron.NewTRX(burntTrxForBandwidth.Add(burntTrxForEnergy).Div(decimal.NewFromInt(1e6))).ToDecimal(),
		CurrentTxStatus:   status,
		TransferID:        s.transfer.ID,
		TxHash:            txHash,
	})
}

func (s *FSM) prepareTransferTransactionTypeByStep() (*models.TransferTransactionType, error) {
	switch s.wf.CurrentStep().Name {
	case stepActivateWallet, stepActiveWalletResources, stepActivateWalletBurnTRX:
		return utils.Pointer(models.TransferTransactionTypeAccountActivation), nil
	case stepSendTRXForBurn:
		return utils.Pointer(models.TransferTransactionTypeSendBurnBaseAsset), nil
	case stepDelegateResources:
		return utils.Pointer(models.TransferTransactionTypeDelegateResources), nil
	case stepReclaimResources, stepReclaimOnError, stepWaitingReclaimRresourcesConfirmations, stepWaitingForTheFirstConfirmation:
		return utils.Pointer(models.TransferTransactionTypeReclaimResources), nil
	case stepSending:
		return utils.Pointer(models.TransferTransactionTypeTransfer), nil
	}

	return nil, fmt.Errorf("cant resolve transfer transaction type by step: %s", s.wf.CurrentStep().Name)
}
