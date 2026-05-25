package taskmanager

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
	"github.com/dv-net/mx/logger"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
)

const JobKindWebhookWaitingConfirmations = "waiting_confirmations"

type WebhookWaitingConfirmationsArgs struct {
	Blockchain             wconstants.BlockchainType `json:"blockchain"`
	Hash                   string                    `json:"hash"`
	Address                string                    `json:"address"`
	EventBlockchainUniqKey string                    `json:"event_blockchain_uniq_key"`
	WebhookKind            models.WebhookKind        `json:"webhook_kind"`
	WalletType             constants.WalletType      `json:"wallet_type"`
	OwnerID                uuid.UUID                 `json:"owner_id"`
	ExternalWalletID       *string                   `json:"external_wallet_id,omitempty"`
	IsSystem               bool                      `json:"is_system"`
}

func (s WebhookWaitingConfirmationsArgs) Validate() error {
	if !s.Blockchain.Valid() {
		return fmt.Errorf("blockchain %s is invalid", s.Blockchain)
	}

	if s.Hash == "" {
		return fmt.Errorf("hash is empty")
	}

	if s.Address == "" {
		return fmt.Errorf("address is empty")
	}

	if s.EventBlockchainUniqKey == "" {
		return fmt.Errorf("event blockchain uniq key is empty")
	}

	if !s.WebhookKind.Valid() {
		return fmt.Errorf("webhook kind %s is invalid", s.WebhookKind)
	}

	if !s.WalletType.Valid() {
		return fmt.Errorf("wallet type %s is invalid", s.WalletType)
	}

	if s.OwnerID == uuid.Nil {
		return fmt.Errorf("owner id is empty")
	}

	return nil
}

// Kind
func (WebhookWaitingConfirmationsArgs) Kind() string { return JobKindWebhookWaitingConfirmations }

func (s *WebhookWaitingConfirmationsWorker) verifyEVMDepositBalanceDelta(
	ctx context.Context,
	job *river.Job[WebhookWaitingConfirmationsArgs],
	tx *transactionsv2.Transaction,
	event *transactionsv2.Event,
) error {
	assetIdentifier := event.GetAssetIdentifier()
	if assetIdentifier == "" {
		return nil
	}

	expectedAmount, err := decimal.NewFromString(event.GetValue())
	if err != nil || !expectedAmount.IsPositive() {
		return nil
	}

	blockHeight := tx.GetBlockHeight()
	if blockHeight == 0 {
		return nil
	}

	balanceBefore, err := s.bs.EProxy().AddressBalanceAt(ctx, job.Args.Address, assetIdentifier, job.Args.Blockchain, blockHeight-1)
	if err != nil {
		return fmt.Errorf("get balance before deposit block %d: %w", blockHeight-1, err)
	}

	balanceAfter, err := s.bs.EProxy().AddressBalanceAt(ctx, job.Args.Address, assetIdentifier, job.Args.Blockchain, blockHeight)
	if err != nil {
		return fmt.Errorf("get balance after deposit block %d: %w", blockHeight, err)
	}

	s.logger.Debugf("deposit balance delta raw: address=%s asset=%s block=%d before=%s after=%s expected=%s",
		job.Args.Address, assetIdentifier, blockHeight, balanceBefore.String(), balanceAfter.String(), expectedAmount.String())

	delta := balanceAfter.Sub(balanceBefore)
	if delta.LessThan(expectedAmount) {
		return fmt.Errorf("deposit balance delta mismatch: address=%s asset=%s block=%d expected=%s actual_delta=%s",
			job.Args.Address, assetIdentifier, blockHeight, expectedAmount.String(), delta.String())
	}
	s.logger.Debugf("deposit balance delta verified: address=%s asset=%s block=%d expected=%s actual_delta=%s",
		job.Args.Address, assetIdentifier, blockHeight, expectedAmount.String(), delta.String())

	return nil
}

type WebhookWaitingConfirmationsWorker struct {
	logger logger.Logger
	river.WorkerDefaults[WebhookWaitingConfirmationsArgs]

	bs baseservices.IBaseServices
}

func (s *WebhookWaitingConfirmationsWorker) Work(ctx context.Context, job *river.Job[WebhookWaitingConfirmationsArgs]) error {
	if err := job.Args.Validate(); err != nil {
		return fmt.Errorf("validate args: %w", err)
	}

	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, job.Args.Blockchain, job.Args.Hash)
	if err != nil {
		return fmt.Errorf("get transaction info for blockchain %s and hash %s: %w", job.Args.Blockchain, job.Args.Hash, err)
	}

	if len(tx.Events) == 0 {
		return fmt.Errorf("transaction %s has no events", job.Args.Hash)
	}

	var event *transactionsv2.Event
	for _, e := range tx.Events {
		if e.BlockchainUniqKey == nil {
			continue
		}
		if *e.BlockchainUniqKey == job.Args.EventBlockchainUniqKey {
			event = e
			break
		}
	}

	if event == nil {
		return fmt.Errorf("event not found by blockchain uniq key %s", job.Args.EventBlockchainUniqKey)
	}

	confirmationsTimeout := constants.ConfirmationsTimeout(job.Args.Blockchain, tx.Confirmations)
	if confirmationsTimeout > 0 {
		return river.JobSnooze(confirmationsTimeout)
	}

	if job.Args.Blockchain.IsEVM() && job.Args.WebhookKind == models.WebhookKindDeposit {
		if err := s.verifyEVMDepositBalanceDelta(ctx, job, tx, event); err != nil {
			return err
		}
	}

	transactionData := webhooks.TransactionData{
		Hash:          job.Args.Hash,
		Confirmations: tx.Confirmations,
		Fee:           &tx.Fee,
	}

	if tx.Status != "" {
		transactionData.Status = &tx.Status
	}

	if tx.CreatedAt != nil {
		transactionData.CreatedAt = tx.CreatedAt.AsTime()
	}

	transactionEventData := webhooks.TransactionEventData{
		AddressFrom:       event.AddressFrom,
		AddressTo:         event.AddressTo,
		Value:             event.Value,
		AssetIdentify:     event.AssetIdentifier,
		BlockchainUniqKey: event.BlockchainUniqKey,
	}

	createParams, err := s.bs.Webhooks().EventTransactionCreateParams(ctx, webhooks.EventTransactionCreateParamsData{
		Blockchain:       job.Args.Blockchain,
		Tx:               transactionData,
		Event:            transactionEventData,
		WebhookKind:      job.Args.WebhookKind,
		WebhookStatus:    models.WebhookEventStatusCompleted,
		WalletType:       job.Args.WalletType,
		OwnerID:          job.Args.OwnerID,
		ExternalWalletID: job.Args.ExternalWalletID,
		IsSystem:         job.Args.IsSystem,
	})
	if err != nil {
		return fmt.Errorf("get create params: %w", err)
	}

	if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{createParams}); err != nil {
		return fmt.Errorf("batch create webhooks: %w", err)
	}

	return nil
}
