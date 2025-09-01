package models

import "fmt"

/*

	WebhookStatus represents the status of a webhook

*/

type WebhookStatus string

const (
	WebhookStatusNew  WebhookStatus = "new"
	WebhookStatusSent WebhookStatus = "sent"
)

// String returns the webhook status as a string
func (w WebhookStatus) String() string { return string(w) }

// Valid
func (w WebhookStatus) Valid() bool {
	switch w {
	case WebhookStatusNew, WebhookStatusSent:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (w *WebhookStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*w = WebhookStatus(s)
	case string:
		*w = WebhookStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for WebhookStatus: %T", src)
	}
	return nil
}

/*

	WebhookKind represents the kind of a webhook

*/

type WebhookKind string

const (
	WebhookKindTransfer       WebhookKind = "transfer"
	WebhookKindDeposit        WebhookKind = "deposit"
	WebhookKindTransferStatus WebhookKind = "transfer_status"
)

// String returns the webhook kind as a string
func (w WebhookKind) String() string { return string(w) }

// Valid
func (w WebhookKind) Valid() bool {
	switch w {
	case WebhookKindTransfer,
		WebhookKindDeposit,
		WebhookKindTransferStatus:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (w *WebhookKind) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*w = WebhookKind(s)
	case string:
		*w = WebhookKind(s)
	default:
		return fmt.Errorf("unsupported scan type for WebhookKind: %T", src)
	}
	return nil
}

/*

	WebhookEventStatus represents the status of a webhook event (payload)

*/

type WebhookEventStatus string

const (
	WebhookEventStatusWaitingConfirmations WebhookEventStatus = "waiting_confirmations"
	WebhookEventStatusInMempool            WebhookEventStatus = "in_mempool"
	WebhookEventStatusCompleted            WebhookEventStatus = "completed"
	WebhookEventStatusFailed               WebhookEventStatus = "failed"
)

// String returns the webhook event status as a string
func (w WebhookEventStatus) String() string { return string(w) }

// Valid
func (w WebhookEventStatus) Valid() bool {
	switch w {
	case WebhookEventStatusWaitingConfirmations,
		WebhookEventStatusCompleted,
		WebhookEventStatusFailed,
		WebhookEventStatusInMempool:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (w *WebhookEventStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*w = WebhookEventStatus(s)
	case string:
		*w = WebhookEventStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for WebhokEventStatus: %T", src)
	}
	return nil
}
