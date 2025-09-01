package fsmbtc

import (
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/shopspring/decimal"
)

var assetDecimals = decimal.NewFromInt(btc.AssetDecimals)

const (
	stageBeforeSending = "before_sending"
	stageSending       = "sending"
	stageAfterSending  = "after_sending"
)

const (
	stepValidateRequest                = "validate_request"
	stepSending                        = "sending"
	stepWaitingInMempool               = "waiting_in_mempool"
	stepWaitingForTheFirstConfirmation = "waiting_for_the_first_confirmation"
	stepWaitingConfirmations           = "waiting_confirmations"
	stepSendSuccessEvent               = "send_success_event"
)
