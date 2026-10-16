package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	transferv1 "github.com/dv-net/dv-processing/api/processing/transfer/v1"
	"github.com/dv-net/dv-processing/api/processing/transfer/v1/transferv1connect"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/transfers"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
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

func (s *transfersServer) Name() string { return "transfers-server" }

func (s *transfersServer) RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return transferv1connect.NewTransferServiceHandler(s, opts...)
}

// Create - creates a new transfer
func (s *transfersServer) Create(
	ctx context.Context,
	req *connect.Request[transferv1.CreateRequest],
) (*connect.Response[transferv1.CreateResponse], error) {
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

	purpose, err := models.ConvertTransferPurpose(req.Msg.Purpose)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
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
		Purpose:         purpose,
		SourceTxHash:    req.Msg.SourceTxHash,
		TOTP:            req.Msg.GetTotp(),
	}

	newTransfer, err := s.bs.Transfers().Create(ctx, params)
	if err != nil {
		// every known rpc code is passed to the client in the rpc-code metadata
		rpcError, ok := rpccode.IsRPCError(err)
		if ok {
			rpcCode, err := rpccode.NewConnectError(connect.CodeInternal, err)
			s.logger.Debug(
				"processing code status",
				"grpc_code", connect.CodeOf(rpcError.Error).String(),
				"rpc_code", rpcCode,
			)
			return nil, err
		}
		s.logger.Errorf("failed to create transfer: %s", err.Error())
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

// Cancel - cancels a scheduled transfer before its execute_after is reached
func (s *transfersServer) Cancel(
	ctx context.Context,
	req *connect.Request[transferv1.CancelRequest],
) (*connect.Response[transferv1.CancelResponse], error) {
	ownerID, err := uuid.Parse(req.Msg.OwnerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid owner id"))
	}

	transfer, err := s.bs.Transfers().Cancel(ctx, ownerID, req.Msg.RequestId,
		func(ctx context.Context, transfer *models.Transfer, opts ...repos.Option) error {
			params, err := s.bs.Webhooks().EventTransferStatusCreateParams(ctx, webhooks.EventTransferStatusCreateParamsData{
				TransferID: transfer.ID,
				OwnerID:    transfer.OwnerID,
				Status:     constants.TransferStatusCanceled,
			})
			if err != nil {
				return fmt.Errorf("get event transfer status create params: %w", err)
			}

			if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{params}, opts...); err != nil {
				return fmt.Errorf("create canceled event: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, s.scheduleError("cancel transfer", err)
	}

	pbItem, err := transfer.ToPb()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&transferv1.CancelResponse{Item: pbItem}), nil
}

// ExecuteNow - executes a scheduled transfer immediately, confirmed by owner's TOTP
func (s *transfersServer) ExecuteNow(
	ctx context.Context,
	req *connect.Request[transferv1.ExecuteNowRequest],
) (*connect.Response[transferv1.ExecuteNowResponse], error) {
	ownerID, err := uuid.Parse(req.Msg.OwnerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid owner id"))
	}

	transfer, err := s.bs.Transfers().ExecuteNow(ctx, ownerID, req.Msg.RequestId, req.Msg.Totp)
	if err != nil {
		return nil, s.scheduleError("execute transfer now", err)
	}

	pbItem, err := transfer.ToPb()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&transferv1.ExecuteNowResponse{Item: pbItem}), nil
}

// CheckRefund - checks that a refund can be created without creating it
func (s *transfersServer) CheckRefund(
	ctx context.Context,
	req *connect.Request[transferv1.CheckRefundRequest],
) (*connect.Response[transferv1.CheckRefundResponse], error) {
	ownerID, err := uuid.Parse(req.Msg.OwnerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid owner id"))
	}

	blockchain, err := models.ConvertBlockchainType(req.Msg.Blockchain)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	amount, err := decimal.NewFromString(req.Msg.Amount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid amount: %w", err))
	}

	result, err := s.bs.Transfers().CheckRefund(ctx, transfers.CreateTransferRequest{
		OwnerID:         ownerID,
		Blockchain:      blockchain,
		FromAddresses:   []string{req.Msg.FromAddress},
		ToAddresses:     []string{req.Msg.ToAddress},
		AssetIdentifier: req.Msg.AssetIdentifier,
		Amount:          decimal.NullDecimal{Decimal: amount, Valid: true},
		SourceTxHash:    &req.Msg.SourceTxHash,
	})
	if err != nil {
		return nil, s.scheduleError("check refund", err)
	}

	res := &transferv1.CheckRefundResponse{
		FeeFromAmount: result.FeeFromAmount,
		ToSender:      result.ToSender,
	}

	if result.EstimatedFee != nil {
		fee := result.EstimatedFee.String()
		res.EstimatedFee = &fee
	}

	return connect.NewResponse(res), nil
}

// scheduleError converts errors of Cancel / ExecuteNow / CheckRefund, rpc codes are passed in the rpc-code metadata
func (s *transfersServer) scheduleError(action string, err error) error {
	rpcError, ok := rpccode.IsRPCError(err)
	if !ok {
		s.logger.Errorf("failed to %s: %s", action, err.Error())
		return connect.NewError(connect.CodeInternal, err)
	}

	connectCode := connect.CodeFailedPrecondition
	if rpcError.Code == rpccode.RPCCodeTwoFactorRequired || rpcError.Code == rpccode.RPCCodeTwoFactorInvalid {
		connectCode = connect.CodeInvalidArgument
	}

	_, connectErr := rpccode.NewConnectError(connectCode, err)
	return connectErr
}

// GetByRequestID - gets a transfer by id
func (s *transfersServer) GetByRequestID(ctx context.Context, req *connect.Request[transferv1.GetByRequestIDRequest]) (*connect.Response[transferv1.GetByRequestIDResponse], error) {
	transfer, err := s.bs.Transfers().GetByRequestID(ctx, req.Msg.RequestId)
	if err != nil {
		// not found is a definite answer: clients rely on it to know that a transfer was not created
		if errors.Is(err, storecmn.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
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
