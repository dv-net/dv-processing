package btc_test

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTxBuilder(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)
	assert.NotNil(t, builder)
}

func TestAddInput(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)

	input := btc.TxInput{
		Hash:     "0c1a00189296a937e9bbee4114eb4283edb700834624924e48ca44d04281d712",
		Sequence: 0,
		Amount:   1000,
	}

	err := builder.AddInput(input)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(builder.MsgTx().TxIn))
}

func TestAddOutput(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)

	address := "bc1q6uwkfj82nuhnz30zxk25zqad5xf8qqaayteh55"
	amount := decimal.NewFromInt(1000)

	err := builder.AddOutput(address, amount)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(builder.MsgTx().TxOut))
}

func TestSignTx(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)

	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	input := btc.TxInput{
		PrivateKey: privKey,
		PkScript:   "0014a04c9b4d0d3b7c80b327b3e81f148e892cb22718",
		Hash:       "55cce9fb5866aa592695b8f3f91bea1c64f6c0cb6fec513e6e67f020aa6c27bf",
		Sequence:   0,
		Amount:     1000,
	}

	err = builder.AddInput(input)
	require.NoError(t, err)

	err = builder.SignTx()
	assert.NoError(t, err)
}

func TestCalculateTxSize(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)

	walletSDK := btc.NewWalletSDK(chainParams)

	addrData, err := walletSDK.GenerateAddress(btc.AddressTypeP2TR, mnemonic, passphrase, 0)
	require.NoError(t, err)

	builder.AddInput(btc.TxInput{
		PrivateKey: addrData.PrivateKey,
		PkScript:   "0014a04c9b4d0d3b7c80b327b3e81f148e892cb22718",
		Hash:       "0c1a00189296a937e9bbee4114eb4283edb700834624924e48ca44d04281d712",
		Sequence:   0,
		Amount:     1000,
	})

	builder.AddOutput("bc1q6uwkfj82nuhnz30zxk25zqad5xf8qqaayteh55", decimal.NewFromInt(1000))

	err = builder.SignTx()
	require.NoError(t, err)

	feePerByte := decimal.NewFromInt(10)
	sizeData := builder.CalculateTxSize(feePerByte)

	fmt.Println(sizeData.TxFullSize)

	assert.True(t, sizeData.TxFullSize.GreaterThan(decimal.Zero))
	assert.True(t, sizeData.TotalFee.GreaterThan(decimal.Zero))
}

func TestEmulateTxSize(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)

	walletSDK := btc.NewWalletSDK(chainParams)

	addrData, err := walletSDK.GenerateAddress(btc.AddressTypeP2WPKH, mnemonic, passphrase, 0)
	require.NoError(t, err)

	builder.AddInput(btc.TxInput{
		PrivateKey: addrData.PrivateKey,
		PkScript:   "0014a04c9b4d0d3b7c80b327b3e81f148e892cb22718",
		Hash:       "0c1a00189296a937e9bbee4114eb4283edb700834624924e48ca44d04281d712",
		Sequence:   0,
		Amount:     1000000,
	})

	builder.AddOutput("bc1q6uwkfj82nuhnz30zxk25zqad5xf8qqaayteh55", decimal.NewFromInt(1000))

	err = builder.SignTx()
	require.NoError(t, err)

	feePerByte := decimal.NewFromInt(10)
	sizeData, err := builder.EmulateTxSize(feePerByte)
	assert.NoError(t, err)

	fmt.Println(sizeData.TotalFee)

	assert.True(t, sizeData.TxFullSize.GreaterThan(decimal.Zero))
	assert.True(t, sizeData.TotalFee.GreaterThan(decimal.Zero))
}

// TestEmulateTx this test does not working
func TestEmulateTx(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	builder := btc.NewTxBuilder(chainParams)

	addresses, err := generateTestAddresses(1, btc.AddressTypeP2WPKH)
	require.NoError(t, err)

	toAddresses, err := generateTestAddresses(1, btc.AddressTypeP2WPKH)
	require.NoError(t, err)

	var totalUTXOAmount decimal.Decimal
	for _, addr := range addresses {
		data, err := generateRandomUTXOForAddress(addr, 1)
		require.NoError(t, err)

		for _, utxo := range data {
			err := builder.AddInput(utxo)
			require.NoError(t, err)
			totalUTXOAmount = totalUTXOAmount.Add(decimal.NewFromInt(utxo.Amount))
		}
	}

	if !totalUTXOAmount.IsPositive() {
		t.Fatalf("total utxo amount is less than or equal to zero: %s", totalUTXOAmount)
	}

	t.Logf("total utxo amount: %s", totalUTXOAmount)

	// set output
	err = builder.AddOutput(toAddresses[0].Address.String(), totalUTXOAmount)
	require.NoError(t, err)

	// emulate transaction and calculate fee
	txSizeData, err := builder.EmulateTxSize(decimal.NewFromFloat(30.2))
	require.NoError(t, err)

	t.Logf("transaction data: %+v", txSizeData)
}
