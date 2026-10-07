package fsmbtc

import (
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/shopspring/decimal"
)

var assetDecimals = decimal.NewFromInt(btc.AssetDecimals)

// dustThreshold is the minimum output value in base units accepted by the network.
// P2PKH dust threshold at the default dust relay fee (3 sat/vB); covers all address types.
var dustThreshold = decimal.NewFromInt(546)

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
