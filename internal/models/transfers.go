package models

import (
	"fmt"

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
	case constants.TransferStatusCanceled:
		return transferv1.Status_STATUS_CANCELED
	default:
		return transferv1.Status_STATUS_UNSPECIFIED
	}
}

// ConvertTransferPurposeToPb converts a TransferPurpose to a TransferPurpose protobuf message
func ConvertTransferPurposeToPb(purpose constants.TransferPurpose) transferv1.TransferPurpose {
	switch purpose {
	case constants.TransferPurposeWithdrawal:
		return transferv1.TransferPurpose_TRANSFER_PURPOSE_WITHDRAWAL
	case constants.TransferPurposeRefund:
		return transferv1.TransferPurpose_TRANSFER_PURPOSE_REFUND
	default:
		return transferv1.TransferPurpose_TRANSFER_PURPOSE_UNSPECIFIED
	}
}

// ConvertTransferPurpose converts transferv1.TransferPurpose to constants.TransferPurpose.
func ConvertTransferPurpose(purpose transferv1.TransferPurpose) (constants.TransferPurpose, error) {
	switch purpose {
	case transferv1.TransferPurpose_TRANSFER_PURPOSE_UNSPECIFIED:
		return constants.TransferPurposeDefault, nil
	case transferv1.TransferPurpose_TRANSFER_PURPOSE_WITHDRAWAL:
		return constants.TransferPurposeWithdrawal, nil
	case transferv1.TransferPurpose_TRANSFER_PURPOSE_REFUND:
		return constants.TransferPurposeRefund, nil
	}

	return "", fmt.Errorf("unsupported transfer purpose: %s", purpose)
}

// ConvertTransferWalletTypeToPb converts a WalletType to a transfer WalletType protobuf message
func ConvertTransferWalletTypeToPb(walletType constants.WalletType) transferv1.WalletType {
	switch walletType {
	case constants.WalletTypeHot:
		return transferv1.WalletType_WALLET_TYPE_HOT
	case constants.WalletTypeCold:
		return transferv1.WalletType_WALLET_TYPE_COLD
	case constants.WalletTypeProcessing:
		return transferv1.WalletType_WALLET_TYPE_PROCESSING
	case constants.WalletTypeExternal:
		return transferv1.WalletType_WALLET_TYPE_EXTERNAL
	default:
		return transferv1.WalletType_WALLET_TYPE_UNSPECIFIED
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
		Purpose:         ConvertTransferPurposeToPb(t.Purpose),
		WalletFromType:  ConvertTransferWalletTypeToPb(t.WalletFromType),
	}

	// wallet to type is not set for transfers created before it was stored
	if t.WalletToType != nil {
		res.WalletToType = ConvertTransferWalletTypeToPb(*t.WalletToType)
	}

	if t.SourceTxHash.Valid && t.SourceTxHash.String != "" {
		res.SourceTxHash = &t.SourceTxHash.String
	}

	if t.ExecuteAfter.Valid && !t.ExecuteAfter.Time.IsZero() {
		res.ExecuteAfter = timestamppb.New(t.ExecuteAfter.Time)
	}

	if t.TwoFactorConfirmedAt.Valid && !t.TwoFactorConfirmedAt.Time.IsZero() {
		res.TwoFactorConfirmedAt = timestamppb.New(t.TwoFactorConfirmedAt.Time)
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

// IsRefund returns true for refunds. Refunds are self-funded: they never spend
// coins or resources of the processing wallet, the fee is paid by the hot wallet itself.
func (t Transfer) IsRefund() bool {
	return t.Purpose == constants.TransferPurposeRefund
}

// FeeFromAmount returns true when the network fee is subtracted from the transfer amount:
// refunds of the native coin of the blockchain.
func (t Transfer) FeeFromAmount() bool {
	return t.IsRefund() && t.AssetIdentifier == t.Blockchain.GetAssetIdentifier()
}
