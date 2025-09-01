package fsmevm

const (
	stageBeforeSending = "before_sending"
	stageSending       = "sending"
	stageAfterSending  = "after_sending"
)

const (
	stepValidateRequest                          = "validate_request"
	stepSendBaseAssetForBurn                     = "send_base_asset_for_burn"
	stepWaitingSendBaseAssetForBurnConfirmations = "waiting_send_base_asset_for_burn_confirmations"
	stepSending                                  = "sending"
	stepWaitingForTheFirstConfirmation           = "waiting_for_the_first_confirmation"
	stepWaitingConfirmations                     = "waiting_confirmations"
	stepSendSuccessEvent                         = "send_success_event"
)

var exitOnFailedSteps = map[string][]string{
	stageBeforeSending: {stepValidateRequest, stepSendBaseAssetForBurn},
	stageSending:       {stepSending},
}

const (
	SendBurnBaseAsset       = "send_burn_base_asset"
	SendBurnBaseAssetTxHash = "send_burn_base_asset_tx_hash"
)
