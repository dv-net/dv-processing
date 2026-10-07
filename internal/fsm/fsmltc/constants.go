package fsmltc

import (
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/shopspring/decimal"
)

var assetDecimals = decimal.NewFromInt(ltc.AssetDecimals)

// dustThreshold is the minimum output value in base units accepted by the network.
// P2PKH dust threshold at the Litecoin Core default dust relay fee (30 litoshi/vB).
var dustThreshold = decimal.NewFromInt(5460)

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
