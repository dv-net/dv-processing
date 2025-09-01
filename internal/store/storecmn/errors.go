package storecmn

import "errors"

var (
	ErrNotFound                    = errors.New("not found")
	ErrEmptyHash                   = errors.New("empty hash")
	ErrEmptyID                     = errors.New("empty id")
	ErrEmptyBlockHeight            = errors.New("empty block height")
	ErrEmptyAddress                = errors.New("empty address")
	ErrEmptyCreatedAt              = errors.New("empty created_at")
	ErrEmptyTransactionHash        = errors.New("empty transaction hash")
	ErrEmptyTransactionBlockNumber = errors.New("empty transaction block number")
	ErrEmptyFromAddress            = errors.New("empty from address")
	ErrEmptyToAddress              = errors.New("empty to address")
	ErrEmptyContractAddress        = errors.New("empty contract address")
	ErrEmptyStatus                 = errors.New("empty status")
	ErrEmptyOTP                    = errors.New("empty otp")
	ErrAlreadyExists               = errors.New("already exists")
)
