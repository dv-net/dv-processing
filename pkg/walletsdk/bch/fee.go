package bch

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// EstimateFeePerByte returns the current network fee rate in base units per byte
// estimated by the node. Returns zero if the node does not have an estimate.
func (t *BCH) EstimateFeePerByte() (decimal.Decimal, error) {
	// Bitcoin Cash Node estimatefee takes no arguments, so the typed client call with nblocks can't be used
	raw, err := t.node.RawRequest("estimatefee", nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("estimate fee: %w", err)
	}

	// fee rate is returned in coins per kilobyte, it is negative when there is no estimate
	var feeRate float64
	if err := json.Unmarshal(raw, &feeRate); err != nil {
		return decimal.Zero, fmt.Errorf("unmarshal estimate fee result %s: %w", string(raw), err)
	}

	if feeRate <= 0 {
		return decimal.Zero, nil
	}

	return decimal.NewFromFloat(feeRate).Mul(decimal.NewFromInt(AssetDecimals)).Div(decimal.NewFromInt(1000)), nil
}
