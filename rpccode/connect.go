package rpccode

import (
	"connectrpc.com/connect"
)

// NewConnectError creates a new connectrpc error with the given connect code and error.
func NewConnectError(connectCode connect.Code, err error) (RPCCode, error) {
	var rpcCode RPCCode
	if err == nil {
		return rpcCode, nil
	}

	connectErr := connect.NewError(connectCode, err)
	if rpcErr, ok := IsRPCError(err); ok {
		rpcCode = rpcErr.Code
		connectErr.Meta().Add("rpc-code", rpcErr.Code.String())
	}

	return rpcCode, connectErr
}
