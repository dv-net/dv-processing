package whevents

import (
	"bytes"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
)

// EventTransferStatusPayload
type EventTransferStatusPayload struct {
	Kind               models.WebhookKind            `json:"kind"`
	IsSystem           bool                          `json:"is_system"`
	RequestID          string                        `json:"request_id"`
	Status             constants.TransferStatus      `json:"status"`
	Step               string                        `json:"step"`
	SystemTransactions []*models.TransferTransaction `json:"system_transactions"`
	ErrorMessage       string                        `json:"error_message,omitempty"`
	// Broadcast is true when the transfer transaction was sent to the network (or its result is unknown).
	// A failed transfer with Broadcast may still be mined and must not be repeated automatically
	Broadcast bool `json:"broadcast"`
}

func (p EventTransferStatusPayload) RawMessage() (*bytes.Buffer, error) { return rawMessage(p) }
