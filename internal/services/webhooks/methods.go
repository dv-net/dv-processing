package webhooks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/webhooks/whevents"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/google/uuid"
)

type TransactionData struct {
	Hash          string
	Confirmations uint64
	Status        *string
	Fee           *string
	CreatedAt     time.Time
}

type TransactionEventData struct {
	AddressFrom       *string
	AddressTo         *string
	Value             *string
	AssetIdentify     *string
	BlockchainUniqKey *string
}

type EventTransactionCreateParamsData struct {
	Blockchain       wconstants.BlockchainType
	Tx               TransactionData
	Event            TransactionEventData
	WebhookKind      models.WebhookKind
	WalletType       constants.WalletType
	WebhookStatus    models.WebhookEventStatus
	OwnerID          uuid.UUID
	ExternalWalletID *string
	IsSystem         bool
}

// EventTransactionCreateParams returns create params for a transaction event.
func (s *Service) EventTransactionCreateParams(ctx context.Context, params EventTransactionCreateParamsData) (BatchCreateParams, error) {
	payloadParams := whevents.EventTransactionPayload{
		Kind:          params.WebhookKind,
		IsSystem:      params.IsSystem,
		Blockchain:    params.Blockchain,
		Hash:          params.Tx.Hash,
		Confirmations: params.Tx.Confirmations,
		Status:        params.WebhookStatus,
		WalletType:    params.WalletType,
	}

	if !params.Blockchain.Valid() {
		return BatchCreateParams{}, fmt.Errorf("blockchain %s is invalid", params.Blockchain)
	}

	if params.Tx.Status != nil &&
		*params.Tx.Status == "failed" {
		payloadParams.Status = models.WebhookEventStatusFailed
	}

	if params.Tx.Fee != nil {
		payloadParams.Fee = *params.Tx.Fee
	}

	if !params.Tx.CreatedAt.IsZero() {
		payloadParams.NetworkCreatedAt = params.Tx.CreatedAt
	}

	if params.Event.AddressFrom != nil && *params.Event.AddressFrom != "" {
		payloadParams.FromAddress = *params.Event.AddressFrom
	}

	if params.Event.AddressTo != nil && *params.Event.AddressTo != "" {
		payloadParams.ToAddress = *params.Event.AddressTo
	}

	if params.Event.Value != nil && *params.Event.Value != "" {
		payloadParams.Amount = *params.Event.Value
	}

	if params.Event.BlockchainUniqKey != nil && *params.Event.BlockchainUniqKey != "" {
		payloadParams.TxUniqKey = *params.Event.BlockchainUniqKey
	}

	if params.Event.AssetIdentify != nil && *params.Event.AssetIdentify != "" {
		payloadParams.ContractAddress = *params.Event.AssetIdentify
	}

	if params.ExternalWalletID != nil && *params.ExternalWalletID != "" {
		payloadParams.ExternalWalletID = *params.ExternalWalletID
	}

	// set request id for transfer
	if payloadParams.Kind == models.WebhookKindTransfer {
		transfer, err := s.transfersService.GetByTxHashAndOwnerID(ctx, params.Tx.Hash, params.OwnerID)
		if err != nil && !errors.Is(err, storecmn.ErrNotFound) {
			return BatchCreateParams{}, fmt.Errorf("get transfer by tx hash and owner id: %w", err)
		}

		if err == nil {
			payloadParams.RequestID = transfer.RequestID
		}
	}

	payload, err := payloadParams.RawMessage()
	if err != nil {
		return BatchCreateParams{}, fmt.Errorf("get raw message for payload: %w", err)
	}

	// get owner
	owner, err := s.ownersService.GetByID(ctx, params.OwnerID)
	if err != nil {
		return BatchCreateParams{}, fmt.Errorf("get owner: %w", err)
	}

	return BatchCreateParams{
		Kind:     params.WebhookKind,
		Status:   models.WebhookStatusNew,
		Payload:  payload.Bytes(),
		ClientID: owner.ClientID,
	}, nil
}

type EventTransferStatusCreateParamsData struct {
	IsSystem     bool
	TransferID   uuid.UUID
	OwnerID      uuid.UUID
	Status       constants.TransferStatus
	Step         string
	ErrorMessage string
}

// EventTransferStatusCreateParams returns create params for a transfer status event.
func (s *Service) EventTransferStatusCreateParams(ctx context.Context, params EventTransferStatusCreateParamsData) (BatchCreateParams, error) {
	if params.TransferID == uuid.Nil {
		return BatchCreateParams{}, storecmn.ErrEmptyHash
	}

	if params.OwnerID == uuid.Nil {
		return BatchCreateParams{}, storecmn.ErrEmptyID
	}

	if !params.Status.Valid() {
		return BatchCreateParams{}, fmt.Errorf("status %s is invalid", params.Status)
	}

	// get transfer
	transfer, err := s.transfersService.GetByID(ctx, params.TransferID)
	if err != nil {
		return BatchCreateParams{}, fmt.Errorf("get transfer by id: %w", err)
	}

	sysTxs, err := s.store.TransferTransactions().GetByTransfer(ctx, transfer.ID)
	if err != nil {
		return BatchCreateParams{}, fmt.Errorf("get transfer transactions: %w", err)
	}

	payloadParams := whevents.EventTransferStatusPayload{
		Kind:               models.WebhookKindTransferStatus,
		IsSystem:           params.IsSystem,
		Status:             params.Status,
		Step:               params.Step,
		ErrorMessage:       params.ErrorMessage,
		SystemTransactions: sysTxs,
		RequestID:          transfer.RequestID,
	}

	payload, err := payloadParams.RawMessage()
	if err != nil {
		return BatchCreateParams{}, fmt.Errorf("get raw message for payload: %w", err)
	}

	// get owner
	owner, err := s.store.Owners().GetByID(ctx, params.OwnerID)
	if err != nil {
		return BatchCreateParams{}, fmt.Errorf("get owner: %w", err)
	}

	return BatchCreateParams{
		Kind:     models.WebhookKindTransferStatus,
		Status:   models.WebhookStatusNew,
		Payload:  payload.Bytes(),
		ClientID: owner.ClientID,
	}, nil
}
