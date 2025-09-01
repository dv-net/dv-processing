package eproxy

import "errors"

var (
	errConnectionResetByPeer   = "connection reset by peer"
	ErrHashIsRequired          = errors.New("hash is required")
	ErrAssetIdentifierRequired = errors.New("asset identifier is required")
	ErrAddressRequired         = errors.New("address is required")
)
