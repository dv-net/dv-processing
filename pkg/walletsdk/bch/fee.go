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

// MinRelayFeePerByte returns the lowest fee rate in base units per byte the node accepts into its mempool:
// the larger of the node relay fee and the current mempool minimum fee.
// Transactions paying less are rejected with "min relay fee not met".
func (t *BCH) MinRelayFeePerByte() (decimal.Decimal, error) {
	var networkInfo struct {
		RelayFee float64 `json:"relayfee"` //nolint:tagliatelle // node rpc field name
	}
	if err := callNodeRaw(t.node.RawRequest, "getnetworkinfo", &networkInfo); err != nil {
		return decimal.Zero, err
	}

	var mempoolInfo struct {
		MempoolMinFee float64 `json:"mempoolminfee"` //nolint:tagliatelle // node rpc field name
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
