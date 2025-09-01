package tron_test

import (
	"context"
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestConverter(t *testing.T) {
	energyFee := int64(420)
	transactionFee := int64(1e3)

	trxValue := tron.NewTRX(decimal.NewFromInt(1))
	require.Equal(t, "1", trxValue.ToDecimal().String())
	require.Equal(t, "1000000", trxValue.ToSUN().String())
	require.Equal(t, "2380.9523809524", trxValue.ToEnergy(energyFee).String())
	require.Equal(t, "1000", trxValue.ToBandwidth(transactionFee).String())

	energyValue := tron.NewEnergy(decimal.NewFromFloat(2380.9523809524))
	require.Equal(t, "1", energyValue.ToTRX(energyFee).ToDecimal().Round(5).String())

	bandwidthValue := tron.NewBandwidth(decimal.NewFromInt(1000))
	require.Equal(t, "1", bandwidthValue.ToTRX(transactionFee).ToDecimal().String())
}

func TestConverterWithNode(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := context.Background()

	tr.Start(ctx)
	defer tr.Stop(ctx)

	chainParams, err := tr.ChainParams(ctx)
	require.NoError(t, err)

	trxValue := tron.NewTRX(decimal.NewFromInt(1))
	require.Equal(t, "1", trxValue.ToDecimal().String())
	require.Equal(t, "1000000", trxValue.ToSUN().String())
	require.Equal(t, "4761.9047619048", trxValue.ToEnergy(chainParams.EnergyFee).String())
	require.Equal(t, "1000", trxValue.ToBandwidth(chainParams.TransactionFee).String())
}

func TestDecimals(t *testing.T) {
	amount := decimal.NewFromFloat(6.4325)
	dec := decimal.NewFromInt(6)
	amount = amount.Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(dec)))

	require.Equal(t, "6432500", amount.String())
}
