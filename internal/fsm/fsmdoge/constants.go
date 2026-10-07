package fsmdoge

import (
	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/shopspring/decimal"
)

var assetDecimals = decimal.NewFromInt(doge.AssetDecimals)

// dustThreshold is the minimum output value in base units accepted by the network.
// Dogecoin Core soft dust limit (0.01 DOGE).
var dustThreshold = decimal.NewFromInt(1000000)

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
