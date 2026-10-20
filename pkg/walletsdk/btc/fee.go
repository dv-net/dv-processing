package btc

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// feeConfTarget is the number of blocks the transaction is expected to be confirmed within
const feeConfTarget = 3

// EstimateFeePerByte returns the current network fee rate in base units per virtual byte
// estimated by the node. Returns zero if the node does not have enough data for the estimate yet.
func (t *BTC) EstimateFeePerByte() (decimal.Decimal, error) {
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

// MinRelayFeePerByte returns the lowest fee rate in base units per byte the node accepts into its mempool:
// the larger of the node relay fee and the current mempool minimum fee.
// Transactions paying less are rejected with "min relay fee not met".
func (t *BTC) MinRelayFeePerByte() (decimal.Decimal, error) {
	var networkInfo struct {
		RelayFee float64 `json:"relayfee"`
	}
	if err := callNodeRaw(t.node.RawRequest, "getnetworkinfo", &networkInfo); err != nil {
		return decimal.Zero, err
	}

	var mempoolInfo struct {
		MempoolMinFee float64 `json:"mempoolminfee"`
	}
	if err := callNodeRaw(t.node.RawRequest, "getmempoolinfo", &mempoolInfo); err != nil {
		return decimal.Zero, err
	}

	// fees are returned in coins per kilobyte
	perKB := decimal.Max(decimal.NewFromFloat(networkInfo.RelayFee), decimal.NewFromFloat(mempoolInfo.MempoolMinFee))

	return perKB.Mul(decimal.NewFromInt(AssetDecimals)).Div(decimal.NewFromInt(1000)), nil
}

func callNodeRaw(rawRequest func(string, []json.RawMessage) (json.RawMessage, error), method string, dst any) error {
	raw, err := rawRequest(method, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}

	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("unmarshal %s result %s: %w", method, string(raw), err)
	}

	return nil
}
