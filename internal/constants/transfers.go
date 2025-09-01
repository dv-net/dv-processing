package constants

type TransferStatus string

const (
	// TransferStatusNew
	//
	// The transfer was created, but not yet taken into processing
	TransferStatusNew TransferStatus = "new"

	// TransferStatusPending
	//
	// The transfer is in anticipation. The transfer is placed in Task Manager and expects when he is taken to work.
	TransferStatusPending TransferStatus = "pending"

	// TransferStatusProcessing
	//
	// Transfer is in processing. In this status, a transfer can activate a wallet, delegate resources, make an ITD translation.
	TransferStatusProcessing TransferStatus = "processing"

	// TransferStatusInMempool
	//
	// The transaction is located in a mempoule (pool of unconfirmed transactions) and expects to include in the block.
	TransferStatusInMempool TransferStatus = "in_mempool"

	// TransferStatusUnconfirmed
	//
	// The transaction expects a sufficient number of confirmations.
	TransferStatusUnconfirmed TransferStatus = "unconfirmed"

	// TransferStatusCompleted
	//
	// The transaction was confirmed and completed (all post -cutting ended, for example, divided resources).
	TransferStatusCompleted TransferStatus = "completed"

	// TransferStatusFailed
	//
	// The transaction ended with an error. It may be associated with insufficient balance, not compliance with the conditions of the smart contract, etc.
	TransferStatusFailed TransferStatus = "failed"

	// TransferStatusFrozen
	//
	// The transaction was frozen (on Hold).
	// can occur when problems of different nature arose in the process of transaction processing,
	// For example, at the time of sending, a network failure and it is not clear whether the transfer occurred or not,
	// and the transaction requires manual intervention.
	TransferStatusFrozen TransferStatus = "frozen"
)

// String
func (t TransferStatus) String() string { return string(t) }

// Valid
func (t TransferStatus) Valid() bool {
	switch t {
	case TransferStatusNew,
		TransferStatusPending,
		TransferStatusProcessing,
		TransferStatusInMempool,
		TransferStatusUnconfirmed,
		TransferStatusCompleted,
		TransferStatusFailed,
		TransferStatusFrozen:
		return true
	}
	return false
}

// AllTransferStatuses
func AllTransferStatuses() []TransferStatus {
	return []TransferStatus{
		TransferStatusNew,
		TransferStatusPending,
		TransferStatusProcessing,
		TransferStatusInMempool,
		TransferStatusUnconfirmed,
		TransferStatusCompleted,
		TransferStatusFailed,
		TransferStatusFrozen,
	}
}
