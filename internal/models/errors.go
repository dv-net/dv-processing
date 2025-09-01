package models

import (
	"errors"
	"strings"
)

var (
	ErrNoRowsInResultSet                        = NewProcessingError("no rows in result set")
	ErrBlockNotFound                            = NewProcessingError("block not found")
	ErrTransactionType        NotificationError = errors.New("define transaction type")
	ErrSendWebhook            NotificationError = errors.New("send webhook")
	ErrBlockProcessingTimeout NotificationError = errors.New("block processing timeout")
	ErrWithdrawalNotFound     NotificationError = errors.New("withdrawal not found")
	ErrWalletCast                               = errors.New("wallet type cast error")
	ErrWalletNotFound                           = errors.New("wallet not found")
	ErrOwnerNotFound                            = errors.New("owner not found")
	ErrClientNotFound                           = errors.New("client not found")
	ErrBlockchainUndefined                      = errors.New("blockchain undefined")
)

type NotificationError error

type ProcessingError string

func NewProcessingError(str string) ProcessingError {
	return ProcessingError(str)
}

func (p ProcessingError) Contains(err error) bool {
	return strings.Contains(err.Error(), string(p))
}

func (p ProcessingError) Error() string {
	return string(p)
}
