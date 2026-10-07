package ltc

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// feeConfTarget is the number of blocks the transaction is expected to be confirmed within
const feeConfTarget = 3

// EstimateFeePerByte returns the current network fee rate in base units per virtual byte
// estimated by the node. Returns zero if the node does not have enough data for the estimate yet.
func (t *LTC) EstimateFeePerByte() (decimal.Decimal, error) {
	res, err := t.node.EstimateSmartFee(feeConfTarget, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("estimate smart fee: %w", err)
	}

	// fee rate is returned in coins per kilobyte, it is negative or absent when there is no estimate
	if res.FeeRate == nil || *res.FeeRate <= 0 {
		return decimal.Zero, nil
	}

	return decimal.NewFromFloat(*res.FeeRate).Mul(decimal.NewFromInt(AssetDecimals)).Div(decimal.NewFromInt(1000)), nil
}
