package models

import (
	transferv1 "github.com/dv-net/dv-processing/api/processing/transfer/v1"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertTransferStatusToPb converts a TransferStatus to a TransferStatus protobuf message
func ConvertTransferStatusToPb(status constants.TransferStatus) transferv1.Status {
	switch status {
	case constants.TransferStatusNew:
		return transferv1.Status_STATUS_NEW
	case constants.TransferStatusProcessing:
		return transferv1.Status_STATUS_PROCESSING
	case constants.TransferStatusInMempool:
		return transferv1.Status_STATUS_IN_MEMPOOL
	case constants.TransferStatusUnconfirmed:
		return transferv1.Status_STATUS_UNCONFIRMED
	case constants.TransferStatusCompleted:
		return transferv1.Status_STATUS_COMPLETED
	case constants.TransferStatusFailed:
		return transferv1.Status_STATUS_FAILED
	case constants.TransferStatusFrozen:
		return transferv1.Status_STATUS_FROZEN
	default:
		return transferv1.Status_STATUS_UNSPECIFIED
	}
}

// ToPb converts a Transfer model to a Transfer protobuf message
func (t *Transfer) ToPb() (*transferv1.Transfer, error) {
	res := &transferv1.Transfer{
		Id:              t.ID.String(),
		Status:          ConvertTransferStatusToPb(t.Status),
		OwnerId:         t.OwnerID.String(),
		RequestId:       t.RequestID,
		Blockchain:      ConvertBlockchainTypeToPb(t.Blockchain),
		FromAddresses:   t.FromAddresses,
		ToAddresses:     t.ToAddresses,
		AssetIdentifier: t.AssetIdentifier,
		WholeAmount:     t.WholeAmount,
	}

	if t.Kind.Valid && t.Kind.String != "" {
		res.Kind = &t.Kind.String
	}

	if t.TxHash.Valid && t.TxHash.String != "" {
		res.TxHash = &t.TxHash.String
	}

	if t.Amount.Valid {
		res.Amount = utils.Pointer(t.Amount.Decimal.String())
	}

	if t.Fee.Valid {
		res.Fee = utils.Pointer(t.Fee.Decimal.String())
	}

	if t.FeeMax.Valid {
		res.FeeMax = utils.Pointer(t.FeeMax.Decimal.String())
	}

	if t.CreatedAt.Valid && !t.CreatedAt.Time.IsZero() {
		res.CreatedAt = timestamppb.New(t.CreatedAt.Time)
	}

	if t.UpdatedAt.Valid && !t.UpdatedAt.Time.IsZero() {
		res.UpdatedAt = timestamppb.New(t.UpdatedAt.Time)
	}

	var err error
	if len(t.StateData) > 0 {
		res.StateData, err = structpb.NewStruct(t.StateData)
		if err != nil {
			return nil, err
		}
	} else {
		res.StateData = new(structpb.Struct)
	}

	workflowSnapshot, err := utils.JSONToStruct[map[string]any](t.WorkflowSnapshot)
	if err != nil {
		return nil, err
	}

	res.WorkflowSnapshot, err = structpb.NewStruct(workflowSnapshot)
	if err != nil {
		return nil, err
	}

	if len(t.WorkflowSnapshot.StepsStates) > 0 && t.WorkflowSnapshot.WorkflowState.IsFailed {
		lastStepState := t.WorkflowSnapshot.StepsStates[len(t.WorkflowSnapshot.StepsStates)-1]
		res.ErrorMessage = &lastStepState.Error
	}

	return res, nil
}

// GetFromAddress returns the first from address
func (t Transfer) GetFromAddress() string {
	if len(t.FromAddresses) == 0 {
		return ""
	}

	return t.FromAddresses[0]
}

// GetToAddress returns the first to address
func (t Transfer) GetToAddress() string {
	if len(t.ToAddresses) == 0 {
		return ""
	}

	return t.ToAddresses[0]
}
