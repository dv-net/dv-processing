package whevents

import (
	"bytes"
	"time"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

// EventTransactionPayload
type EventTransactionPayload struct {
	Kind             models.WebhookKind        `json:"kind"`
	IsSystem         bool                      `json:"is_system"`
	Blockchain       wconstants.BlockchainType `json:"blockchain"`
	Hash             string                    `json:"hash"`
	NetworkCreatedAt time.Time                 `json:"network_created_at"`
	FromAddress      string                    `json:"from_address,omitempty"`
	ToAddress        string                    `json:"to_address,omitempty"`
	Amount           string                    `json:"amount"`
	ContractAddress  string                    `json:"contract_address,omitempty"`
	Status           models.WebhookEventStatus `json:"status"`
	Fee              string                    `json:"fee"`
	Confirmations    uint64                    `json:"confirmations"`
	WalletType       constants.WalletType      `json:"wallet_type"`
	TxUniqKey        string                    `json:"tx_uniq_key,omitempty"`
	ExternalWalletID string                    `json:"external_wallet_id,omitempty"`
	RequestID        string                    `json:"request_id,omitempty"`
}

func (p EventTransactionPayload) RawMessage() (*bytes.Buffer, error) { return rawMessage(p) }
