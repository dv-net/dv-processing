package fsmtron

import (
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

const systemMinConfirmationsCount = 3

const (
	stageBeforeSending    = "before_sending"
	stageSending          = "sending"
	stageAfterSending     = "after_sending"
	stageCompensateOnFail = "compensate"
)

const (
	stepValidateRequest                             = "validate_request"
	stepCheckActivateWallet                         = "check_activate_wallet"
	stepBeforeActivationCheckTransferKind           = "before_activation_check_transfer_kind"
	stepActiveWalletResources                       = "activate_wallet_resources"
	stepWaitingResourcesActivateWalletConfirmations = "waiting_resources_activate_wallet_confirmations"
	stepWaitingExternalActivateWalletConfirmations  = "waiting_external_activate_confirmations"
	stepActivateWalletBurnTRX                       = "activate_wallet_burn_trx"
	stepActivateWallet                              = "activate_wallet"
	stepWaitingActivateWalletConfirmations          = "waiting_activate_confirmations"
	stepBeforeSendingCheckTransferKind              = "before_sending_check_transfer_kind"
	stepSendTRXForBurn                              = "send_trx_for_burn"
	stepWaitingSendTRXForBurnConfirmations          = "waiting_send_trx_for_burn_confirmations"
	stepDelegateResources                           = "delegate_resources"
	stepWaitingDelegateConfirmations                = "waiting_delegate_confirmations"
	stepWaitingExternalDelegateConfirmations        = "waiting_external_delegate_confirmations"
	stepSending                                     = "sending"
	stepWaitingForTheFirstConfirmation              = "waiting_for_the_first_confirmation"
	stepWaitingConfirmations                        = "waiting_confirmations"
	stepAfterSendingCheckTransferKind               = "after_sending_check_transfer_kind"
	stepReclaimResources                            = "reclaim_resources"
	stepWaitingReclaimRresourcesConfirmations       = "waiting_reclaim_resources_confirmations"
	stepSendSuccessEvent                            = "send_success_event"
	stepDetermineCompensationFlow                   = "determine_compensation_flow"
	stepReclaimOnError                              = "reclaim_on_error"
	stepWaitingReclaimOnErrorConfirmation           = "waiting_reclaim_on_error_confirmation"
	stepSendFailureEvent                            = "send_failure_event"
)

var exitOnFailedSteps = map[string][]string{
	stageBeforeSending: {stepValidateRequest, stepCheckActivateWallet, stepActivateWallet, stepActiveWalletResources, stepActivateWalletBurnTRX, stepBeforeSendingCheckTransferKind, stepSendTRXForBurn, stepDelegateResources},
	stageSending:       {stepSending},
	stageAfterSending:  {stepAfterSendingCheckTransferKind, stepReclaimResources},
}

const (
	ActivationOrderID = "activation_order_id"
	BandwidthOrderID  = "bandwidth_delegation_order_id"
	EnergyOrderID     = "energy_delegation_order_id"
)

const (
	ActivateAccountTxHash = "activate_account_tx_hash"
	SendBurntrxTxHash     = "send_burntrx_tx_hash"
	DelegateFromAddress   = "delegate_from"
	ReclaimFromAddress    = "reclaim_from"
)

const (
	DurationTimeResourceDelegation = 120 * time.Second
)

var resourcesToDelegate = []core.ResourceCode{core.ResourceCode_ENERGY, core.ResourceCode_BANDWIDTH}
