package rpccode

import "errors"

type RPCError struct {
	Code  RPCCode
	Error error
}

func IsRPCError(err error) (RPCError, bool) {
	for rpcCode, rpcErr := range RPCCodes {
		if errors.Is(err, rpcErr) {
			return RPCError{
				Code:  rpcCode,
				Error: rpcErr,
			}, true
		}
	}
	return RPCError{}, false
}
