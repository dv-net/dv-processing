package models

import (
	transferv1 "github.com/dv-net/dv-processing/api/processing/transfer/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (tt *TransferTransaction) ToPb() *transferv1.TransferTransaction {
	res := &transferv1.TransferTransaction{
		Id:                tt.ID.String(),
		TransferId:        tt.TransferID.String(),
		TxHash:            tt.TxHash,
		BandwidthAmount:   tt.BandwidthAmount.String(),
		EnergyAmount:      tt.EnergyAmount.String(),
		NativeTokenAmount: tt.NativeTokenAmount.String(),
		NativeTokenFee:    tt.NativeTokenFee.String(),
		TxType:            ConvertTransferTransactionTypeToPb(tt.TxType),
		Status:            ConvertTransferTransactionStatusToPb(tt.Status),
		Step:              tt.Step,
	}

	if tt.CreatedAt.Valid {
		res.CreatedAt = timestamppb.New(tt.CreatedAt.Time)
	}

	if tt.UpdatedAt.Valid {
		res.UpdatedAt = timestamppb.New(tt.UpdatedAt.Time)
	}

	return res
}

func ConvertTransferTransactionStatusToPb(txStatus TransferTransactionsStatus) transferv1.TransferTransactionStatus {
	switch txStatus {
	case TransferTransactionsStatusPending:
		return transferv1.TransferTransactionStatus_TRANSFER_TRANSACTION_STATUS_PENDING
	case TransferTransactionsStatusUnconfirmed:
		return transferv1.TransferTransactionStatus_TRANSFER_TRANSACTION_STATUS_UNCONFIRMED
	case TransferTransactionsStatusConfirmed:
		return transferv1.TransferTransactionStatus_TRANSFER_TRANSACTION_STATUS_CONFIRMED
	case TransferTransactionsStatusFailed:
		return transferv1.TransferTransactionStatus_TRANSFER_TRANSACTION_STATUS_FAILED
	default:
		return transferv1.TransferTransactionStatus_TRANSFER_TRANSACTION_STATUS_UNSPECIFIED
	}
}

func ConvertTransferTransactionTypeToPb(txType TransferTransactionType) transferv1.TransferTransactionType {
	switch txType {
	case TransferTransactionTypeTransfer:
		return transferv1.TransferTransactionType_TRANSFER_TRANSACTION_TYPE_TRANSFER
	case TransferTransactionTypeSendBurnBaseAsset:
		return transferv1.TransferTransactionType_TRANSFER_TRANSACTION_TYPE_SEND_BURN_BASE_ASSET
	case TransferTransactionTypeAccountActivation:
		return transferv1.TransferTransactionType_TRANSFER_TRANSACTION_TYPE_ACCOUNT_ACTIVATION
	case TransferTransactionTypeDelegateResources:
		return transferv1.TransferTransactionType_TRANSFER_TRANSACTION_TYPE_DELEGATE
	case TransferTransactionTypeReclaimResources:
		return transferv1.TransferTransactionType_TRANSFER_TRANSACTION_TYPE_RECLAIM
	default:
		return transferv1.TransferTransactionType_TRANSFER_TRANSACTION_TYPE_UNSPECIFIED
	}
}
