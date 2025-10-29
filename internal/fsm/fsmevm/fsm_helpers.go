package fsmevm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfer_transactions"
	trxv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm/erc20"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
)

// sendFailureEvent
func (s *FSM) sendFailureEvent(ctx context.Context, w *workflow.Workflow, err error, repoOpts ...repos.Option) error {
	var stepName string
	if s.wf.CurrentStep() != nil {
		stepName = s.wf.CurrentStep().Name
	}

	params, err := s.bs.Webhooks().EventTransferStatusCreateParams(ctx, webhooks.EventTransferStatusCreateParamsData{
		TransferID:   s.transfer.ID,
		OwnerID:      s.transfer.OwnerID,
		Status:       constants.TransferStatusFailed,
		Step:         stepName,
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

func (s *FSM) ensureTxInBlockchain(ctx context.Context, txHash string) error {
	// get transaction info
	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, s.transfer.Blockchain, txHash)
	if err != nil {
		if s.transfer.CreatedAt.Valid && time.Since(s.transfer.CreatedAt.Time) > 1*time.Hour {
			if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusFailed); err != nil {
				return newErrFailedTransfer(err)
			}

			return newErrFailedTransfer(fmt.Errorf("transaction %s not found in the blockchain after 1 hour: %w", txHash, err))
		}

		if strings.Contains(err.Error(), "not found") {
			return workflow.NoConsoleError(river.JobSnooze(time.Second))
		}

		return fmt.Errorf("get transaction info: %w", err)
	}

	if tx.GetStatus() != "success" {
		if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusFailed); err != nil {
			return newErrFailedTransfer(err)
		}

		return newErrFailedTransfer(fmt.Errorf("transaction status is %s", tx.GetStatus()))
	}

	// if transaction is not confirmed, snooze the job
	if tx.Confirmations < 1 {
		return workflow.NoConsoleError(river.JobSnooze(
			constants.ConfirmationsTimeoutWithRequired(s.transfer.Blockchain, 1, tx.Confirmations),
		))
	}

	if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusUnconfirmed); err != nil {
		return newErrFailedTransfer(err)
	}

	return nil
}

func (s *FSM) checkTransactionConfirmations(ctx context.Context, txHash string, confirmationsCount uint64) error {
	// get transaction info
	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, s.evm.Blockchain(), txHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return workflow.NoConsoleError(river.JobSnooze(time.Second * 2))
		}
		return fmt.Errorf("get transaction info: %w", err)
	}

	if tx.GetStatus() != "success" {
		if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusFailed); err != nil {
			return fmt.Errorf("update system transaction expenses: %w", err)
		}

		return newErrFailedTransfer(fmt.Errorf("transaction status is not success: %s", tx.GetStatus()))
	}

	// if transaction is not confirmed, snooze the job
	if tx.Confirmations < confirmationsCount {
		if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusUnconfirmed); err != nil {
			return fmt.Errorf("update system transaction expenses: %w", err)
		}

		return workflow.NoConsoleError(river.JobSnooze(
			constants.ConfirmationsTimeoutWithRequired(s.evm.Blockchain(), confirmationsCount, tx.Confirmations),
		))
	}

	return s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusConfirmed)
}

// getBalance returns the balance of the address.
func (s *FSM) getBalance(ctx context.Context, address string, assetIdentifier string) (decimal.Decimal, error) {
	balance, err := s.bs.EProxy().AddressBalance(ctx, address, assetIdentifier, s.evm.Blockchain())
	if err != nil {
		return decimal.Zero, err
	}

	return balance, nil
}

func (s *FSM) getAssetDecimals(ctx context.Context, assetIdentifier string) (int64, error) {
	return s.bs.EProxy().AssetDecimals(ctx, s.evm.Blockchain(), assetIdentifier)
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
	if s.enabledSeedEncryption {
		mnemonic, err = encryption.Decrypt(mnemonic, owner.ID.String())
		if err != nil {
			return nil, fmt.Errorf("decrypt mnemonic: %w", err)
		}
	}

	address, priv, public, err := evm.WalletPubKeyHash(mnemonic, owner.PassPhrase.String, sequence)
	if err != nil {
		return nil, err
	}

	return &walletCreds{
		Address:    address,
		PrivateKey: priv,
		PublicKey:  public,
	}, nil
}

// sendBaseAsset allows to send base asset berween wallets.
//
// Important: Amount in evm. You can use non-integer values, for example, 0.0001 evm. Amount will be converted to WEI automatically.
func (s *FSM) sendBaseAsset(ctx context.Context, wCreds *walletCreds, toAddress string, amount decimal.Decimal, estimateResult *evm.EstimateTransferResult) (*types.Transaction, map[string]any, error) {
	if toAddress == "" {
		return nil, nil, fmt.Errorf("to address is not valid")
	}

	if !amount.IsPositive() {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}

	amountWei := evm.NewUnit(amount, evm.EtherUnitEther).Value(evm.EtherUnitWei)

	// get chain id
	chainID, err := s.evm.Node().ChainID(ctx)
	if err != nil {
		return nil, nil, err
	}

	// get nonce
	nonce, err := s.evm.Node().PendingNonceAt(ctx, common.HexToAddress(wCreds.Address))
	if err != nil {
		return nil, nil, err
	}

	gasLimit := gasLimitByBlockchain(s.evm.Blockchain())

	s.logger.Infow(
		s.stringForBaseAsset("sending %s"),
		"from", wCreds.Address,
		"to", toAddress,
		s.stringForBaseAsset("amount_%s"), amount.String(),
		"amount_wei", amountWei.String(),
		"chain_id", chainID,
		"nonce", nonce,
		"gas_limit", gasLimit,
		"max_fee_per_gas", estimateResult.Estimate.MaxFeePerGas.String(),
		"gas_tip_cap", estimateResult.GasTipCap.String(),
		"total_fee", estimateResult.TotalFeeAmount,
		s.stringForBaseAsset("total_fee_%s"), evm.NewUnit(estimateResult.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).String(),
	)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: estimateResult.Estimate.MaxFeePerGas.BigInt(),
		GasTipCap: estimateResult.GasTipCap.BigInt(),
		Gas:       gasLimit,
		To:        utils.Pointer(common.HexToAddress(toAddress)),
		Value:     amountWei.BigInt(),
		Data:      nil,
	})

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), wCreds.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("sign transaction: %w", err)
	}

	stateData := map[string]any{
		s.stringForBaseAsset("amount_%s"): amount.String(),
		"amount_wei":                      amountWei.String(),
		"nonce":                           nonce,
		"gas_limit":                       gasLimit,
		"estimated_data":                  estimateResult,
	}

	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, signedTx.Hash().Hex(), dbTx); err != nil {
			return fmt.Errorf("create system transaction: %w", err)
		}

		if err = s.evm.Node().SendTransaction(ctx, signedTx); err != nil {
			return fmt.Errorf("send transaction: %w", err)
		}

		return nil
	}); err != nil {
		return nil, nil, err
	}

	return signedTx, stateData, nil
}

func (s *FSM) sendERC20(ctx context.Context, wCreds *walletCreds, contractAddress string, toAddress string, amount decimal.Decimal, decimals int64, estimateResult *evm.EstimateTransferResult) (*types.Transaction, map[string]any, error) {
	if contractAddress == s.evm.Blockchain().GetAssetIdentifier() || contractAddress == "" {
		return nil, nil, fmt.Errorf("contract address is not valid")
	}

	if toAddress == "" {
		return nil, nil, fmt.Errorf("to address is not valid")
	}

	if !amount.IsPositive() {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}

	amount = amount.Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(decimals))))

	// get chain id
	chainID, err := s.evm.Node().ChainID(ctx)
	if err != nil {
		return nil, nil, err
	}

	// get nonce
	nonce, err := s.evm.Node().PendingNonceAt(ctx, common.HexToAddress(wCreds.Address))
	if err != nil {
		return nil, nil, err
	}

	// create auth
	auth, err := bind.NewKeyedTransactorWithChainID(wCreds.PrivateKey, chainID)
	if err != nil {
		return nil, nil, err
	}

	auth.From = common.HexToAddress(wCreds.Address)
	auth.Context = ctx
	auth.Value = big.NewInt(0)
	auth.Nonce = big.NewInt(int64(nonce)) //nolint:gosec
	auth.GasLimit = estimateResult.EstimateGasAmount.BigInt().Uint64()
	auth.GasFeeCap = estimateResult.Estimate.MaxFeePerGas.BigInt()
	auth.GasTipCap = estimateResult.GasTipCap.BigInt()

	s.logger.Infow(
		"sending erc20",
		"from", wCreds.Address,
		"to", toAddress,
		"contract", contractAddress,
		"amount", amount.String(),
		"decimals", decimals,
		"nonce", auth.Nonce.String(),
		"max_fee_per_gas", estimateResult.Estimate.MaxFeePerGas.String(),
		"gas_limit", auth.GasLimit,
		"gas_tip_cap", auth.GasTipCap.String(),
	)

	stateData := map[string]any{
		"contract_address": contractAddress,
		"amount":           amount,
		"decimals":         decimals,
		"nonce":            auth.Nonce.String(),
		"max_fee_per_gas":  estimateResult.Estimate.MaxFeePerGas.String(),
		"gas_limit":        auth.GasLimit,
		"gas_tip_cap":      auth.GasTipCap.String(),
		"estimated_data":   estimateResult,
	}

	transactor, err := erc20.NewERC20Transactor(common.HexToAddress(contractAddress), s.evm.Node())
	if err != nil {
		return nil, nil, fmt.Errorf("create erc20 transactor: %w", err)
	}

	var tx *types.Transaction
	if err = pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		tx, err = transactor.Transfer(auth, common.HexToAddress(toAddress), amount.BigInt())
		if err != nil {
			return fmt.Errorf("create transfer tx: %w", err)
		}

		if err = s.initPendingSystemTransaction(ctx, tx.Hash().Hex(), dbTx); err != nil {
			return fmt.Errorf("create system transaction: %w", err)
		}

		return nil
	}); err != nil {
		return nil, nil, err
	}

	return tx, stateData, nil
}

func (s *FSM) stringForBaseAsset(template string) string {
	return fmt.Sprintf(template, s.evm.Blockchain().GetAssetIdentifier())
}

// initPendingSystemTransaction create system transaction with pending status
func (s *FSM) initPendingSystemTransaction(ctx context.Context, hash string, dbTx pgx.Tx) error {
	txType, err := s.prepareTransferTransactionTypeByStep()
	if err != nil {
		return fmt.Errorf("prepare transfer transaction type: %w", err)
	}

	if _, err = s.st.TransferTransactions(repos.WithTx(dbTx)).Create(ctx, repo_transfer_transactions.CreateParams{
		TransferID: s.transfer.ID,
		TxHash:     hash,
		TxType:     *txType,
		Status:     models.TransferTransactionsStatusPending,
		Step:       s.wf.CurrentStep().Name,
	}); err != nil {
		return fmt.Errorf("create transfer transaction: %w", err)
	}

	return nil
}

// updateSystemTransactionStatus extracts and combines transaction expenses for ab EVM transactions.
// It updates (native_token_amount and native_token_fee) in the transfer_transactions table.
func (s *FSM) updateSystemTransactionStatus(ctx context.Context, tx *trxv2.Transaction, status models.TransferTransactionsStatus) error {
	// Validate input transaction
	if tx == nil {
		return fmt.Errorf("invalid transaction")
	}

	// Get transaction hash and workflow step
	txHash := tx.GetHash()
	if txHash == "" {
		return fmt.Errorf("invalid transaction: empty hash")
	}

	nativeTokenFee, err := decimal.NewFromString(tx.GetFee())
	if err != nil {
		return fmt.Errorf("failed to parse native token fee: %w", err)
	}

	nativeTokenAmount := decimal.Zero
	if tx.GetAssetIdentifier() == s.evm.Blockchain().GetAssetIdentifier() {
		nativeTokenAmount, err = decimal.NewFromString(tx.GetAmount())
		if err != nil {
			return fmt.Errorf("parse native token amount: %w", err)
		}
	}

	return s.st.TransferTransactions().UpdatePendingTxExpense(ctx, repo_transfer_transactions.UpdatePendingTxExpenseParams{
		NativeTokenAmount: nativeTokenAmount,
		NativeTokenFee:    nativeTokenFee,
		CurrentTxStatus:   status,
		TransferID:        s.transfer.ID,
		TxHash:            txHash,
	})
}

func (s *FSM) prepareTransferTransactionTypeByStep() (*models.TransferTransactionType, error) {
	switch s.wf.CurrentStep().Name {
	case stepSendBaseAssetForBurn:
		return utils.Pointer(models.TransferTransactionTypeSendBurnBaseAsset), nil
	case stepSending:
		return utils.Pointer(models.TransferTransactionTypeTransfer), nil
	default:
		return nil, fmt.Errorf("unknown transfer step: %s", s.wf.CurrentStep().Name)
	}
}

// gasLimitByBlockchain returns the gas limit by blockchain.
func gasLimitByBlockchain(blockchain wconstants.BlockchainType) uint64 {
	switch blockchain {
	case wconstants.BlockchainTypeArbitrum:
		return 38000
	default:
		return 21000 // Default gas limit for unknown blockchains
	}
}
