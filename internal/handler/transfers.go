package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	transferv1 "github.com/dv-net/dv-processing/api/processing/transfer/v1"
	"github.com/dv-net/dv-processing/api/processing/transfer/v1/transferv1connect"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/transfers"
	"github.com/dv-net/dv-processing/rpccode"
	"github.com/dv-net/mx/logger"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type transfersServer struct {
	logger logger.Logger

	bs baseservices.IBaseServices

	transferv1connect.UnimplementedTransferServiceHandler
}

func newTransfersServer(
	logger logger.Logger,
	bs baseservices.IBaseServices,
) *transfersServer {
	return &transfersServer{
		logger: logger,
		bs:     bs,
	}
}

func (s transfersServer) Name() string { return "transfers-server" }

func (s *transfersServer) RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return transferv1connect.NewTransferServiceHandler(s, opts...)
}

// Create - creates a new transfer
func (s *transfersServer) Create(ctx context.Context, req *connect.Request[transferv1.CreateRequest]) (*connect.Response[transferv1.CreateResponse], error) {
	ownerID, err := uuid.Parse(req.Msg.OwnerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid owner id"))
	}

	blockchain, err := models.ConvertBlockchainType(req.Msg.Blockchain)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var amount decimal.NullDecimal
	if req.Msg.Amount != nil && *req.Msg.Amount != "" {
		a, err := decimal.NewFromString(*req.Msg.Amount)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid amount: %w", err))
		}

		amount = decimal.NullDecimal{
			Decimal: a,
			Valid:   true,
		}
	}

	var fee decimal.NullDecimal
	if req.Msg.Fee != nil && *req.Msg.Fee != "" {
		f, err := decimal.NewFromString(*req.Msg.Fee)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid fee"))
		}

		fee = decimal.NullDecimal{
			Decimal: f,
			Valid:   true,
		}
	}

	var feeMax decimal.NullDecimal
	if req.Msg.FeeMax != nil && *req.Msg.FeeMax != "" {
		f, err := decimal.NewFromString(*req.Msg.FeeMax)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid max fee"))
		}

		feeMax = decimal.NullDecimal{
			Decimal: f,
			Valid:   true,
		}
	}

	params := transfers.CreateTransferRequest{
		OwnerID:         ownerID,
		RequestID:       req.Msg.RequestId,
		Blockchain:      blockchain,
		FromAddresses:   req.Msg.FromAddresses,
		ToAddresses:     req.Msg.ToAddresses,
		Kind:            req.Msg.Kind,
		AssetIdentifier: req.Msg.AssetIdentifier,
		WholeAmount:     req.Msg.WholeAmount,
		Amount:          amount,
		Fee:             fee,
		FeeMax:          feeMax,
	}

	newTransfer, err := s.bs.Transfers().Create(ctx, params)
	if err != nil {
		rpcCode, err := rpccode.NewConnectError(connect.CodeInternal, err)

		s.logger.Errorf("failed to create transfer [rpc code: %d]: %s", rpcCode, err.Error())

		return nil, err
	}

	pbItem, err := newTransfer.ToPb()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := connect.NewResponse(&transferv1.CreateResponse{
		Item: pbItem,
	})

	return response, nil
}

// GetByRequestID - gets a transfer by id
func (s *transfersServer) GetByRequestID(ctx context.Context, req *connect.Request[transferv1.GetByRequestIDRequest]) (*connect.Response[transferv1.GetByRequestIDResponse], error) {
	transfer, err := s.bs.Transfers().GetByRequestID(ctx, req.Msg.RequestId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbItem, err := transfer.ToPb()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	systemTxs, err := s.bs.Transfers().GetSystemTransactionsByTransfer(ctx, transfer.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbSystemTxs := make([]*transferv1.TransferTransaction, 0, len(systemTxs))
	for _, tx := range systemTxs {
		pbSystemTxs = append(pbSystemTxs, tx.ToPb())
	}

	pbItem.Transactions = pbSystemTxs
	response := connect.NewResponse(&transferv1.GetByRequestIDResponse{
		Item: pbItem,
	})

	return response, nil
}
