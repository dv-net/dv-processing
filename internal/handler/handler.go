package handler

import (
	"github.com/dv-net/dv-processing/api/processing/client/v1/clientv1connect"
	"github.com/dv-net/dv-processing/api/processing/owner/v1/ownerv1connect"
	"github.com/dv-net/dv-processing/api/processing/system/v1/systemv1connect"
	"github.com/dv-net/dv-processing/api/processing/transfer/v1/transferv1connect"
	"github.com/dv-net/dv-processing/api/processing/wallet/v1/walletv1connect"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/mx/logger"
	"github.com/dv-net/mx/transport/connectrpc_transport"
)

type Handler struct {
	ClientsServer interface {
		connectrpc_transport.ConnectRPCService
		clientv1connect.ClientServiceHandler
	}
	OwnersServer interface {
		connectrpc_transport.ConnectRPCService
		ownerv1connect.OwnerServiceHandler
	}
	WalletsServer interface {
		connectrpc_transport.ConnectRPCService
		walletv1connect.WalletServiceHandler
	}
	TransfersServer interface {
		connectrpc_transport.ConnectRPCService
		transferv1connect.TransferServiceHandler
	}
	SystemServer interface {
		connectrpc_transport.ConnectRPCService
		systemv1connect.SystemServiceHandler
	}
}

func New(
	l logger.Logger,
	bs baseservices.IBaseServices,
) *Handler {
	return &Handler{
		ClientsServer:   newClientsServer(bs),
		OwnersServer:    newOwnersServer(bs),
		WalletsServer:   newWalletsServer(bs),
		TransfersServer: newTransfersServer(l, bs),
		SystemServer:    newSystemServer(bs),
	}
}

func (h *Handler) AllServers() []connectrpc_transport.ConnectRPCService {
	return []connectrpc_transport.ConnectRPCService{
		h.ClientsServer,
		h.OwnersServer,
		h.WalletsServer,
		h.TransfersServer,
		h.SystemServer,
	}
}
