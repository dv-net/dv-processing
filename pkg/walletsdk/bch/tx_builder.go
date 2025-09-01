package bch

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
	"github.com/shopspring/decimal"
)

type TxInput struct {
	PrivateKey *bchec.PrivateKey
	PkScript   string
	Hash       string
	Sequence   uint32
	Amount     int64
}

type txOutput struct {
	address string
	amount  decimal.Decimal
}

type TxBuilder struct {
	tx          *wire.MsgTx
	chainParams *chaincfg.Params
	Inputs      []TxInput
	outputs     []txOutput
}

func NewTxBuilder(chainParams *chaincfg.Params) *TxBuilder {
	return &TxBuilder{
		tx:          wire.NewMsgTx(wire.TxVersion),
		chainParams: chainParams,
	}
}

func NewTransactionFromBuilder(builder *TxBuilder) *TxBuilder {
	return &TxBuilder{
		tx:          builder.tx.Copy(),
		chainParams: builder.chainParams,
		Inputs:      builder.Inputs,
		outputs:     builder.outputs,
	}
}

// MsgTx
func (s *TxBuilder) MsgTx() *wire.MsgTx {
	return s.tx
}

func (s *TxBuilder) AddInput(input TxInput) error {
	s.Inputs = append(s.Inputs, input)

	h, err := chainhash.NewHashFromStr(input.Hash)
	if err != nil {
		return fmt.Errorf("cannot make hash from source tx: %w", err)
	}

	outPoint := wire.NewOutPoint(h, input.Sequence)
	for _, in := range s.tx.TxIn {
		if in.PreviousOutPoint.String() == outPoint.String() {
			return ErrInputAlreadyUsed
		}
	}

	txIn := wire.NewTxIn(outPoint, nil)

	// Add RBF flag
	// More details: https://learnmeabitcoin.com/technical/transaction/input/sequence/#replace-by-fee-usage
	txIn.Sequence = 0xfffffffd
	s.tx.AddTxIn(txIn)

	return nil
}

func (s *TxBuilder) AddOutput(address string, amount decimal.Decimal) error {
	for _, out := range s.outputs {
		if out.address == address {
			return ErrOutputAlreadyExists
		}
	}

	address = strings.TrimPrefix(address, s.chainParams.CashAddressPrefix+":")

	addr, err := bchutil.DecodeAddress(address, s.chainParams)
	if err != nil {
		return fmt.Errorf("decode address %s", address)
	}

	if address != addr.String() {
		return fmt.Errorf("addresses is not equal: %s != %s", address, addr.String())
	}

	pk, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return fmt.Errorf("PayToAddrScript error: %w", err)
	}

	s.tx.AddTxOut(wire.NewTxOut(amount.IntPart(), pk, wire.TokenData{}))
	s.outputs = append(s.outputs, txOutput{address: address, amount: amount})

	return nil
}

// SignTx
func (s *TxBuilder) SignTx() error {
	if len(s.Inputs) == 0 {
		return fmt.Errorf("no inputs to sign")
	}

	for idx, input := range s.Inputs {
		if err := s.signTxIn(idx, input.PrivateKey, input.PkScript, input.Amount); err != nil {
			return fmt.Errorf("cannot sign input: %w", err)
		}
	}
	return nil
}

func (s *TxBuilder) signTxIn(index int, privateKey *bchec.PrivateKey, sourcePkScript string, amount int64) error {
	// Decoding the HEX-string of the original script
	srcScript, err := hex.DecodeString(sourcePkScript)
	if err != nil {
		return fmt.Errorf("invalid sourcePkScript: %w", err)
	}

	// We determine the closure to obtain a private key.
	getKey := txscript.KeyClosure(func(_ bchutil.Address) (*bchec.PrivateKey, bool, error) {
		return privateKey, true, nil
	})

	// We determine the short circuit to obtain the script.
	getScript := txscript.ScriptClosure(func(_ bchutil.Address) ([]byte, error) {
		return srcScript, nil
	})

	// We sign the input of the transaction using Sighashall.
	sigScript, err := txscript.SignTxOutput(s.chainParams, s.tx, index, amount,
		srcScript, txscript.SigHashAll, getKey, getScript, s.tx.TxIn[index].SignatureScript)
	if err != nil {
		return fmt.Errorf("failed to sign input: %w", err)
	}

	// Install the generated signature script.
	s.tx.TxIn[index].SignatureScript = sigScript

	// We check that the signature is correct.
	vm, err := txscript.NewEngine(srcScript, s.tx, index, txscript.StandardVerifyFlags, nil, nil, nil, amount)
	if err != nil {
		return fmt.Errorf("failed to create script engine: %w", err)
	}

	if err = vm.Execute(); err != nil {
		return fmt.Errorf("failed to execute script: %w", err)
	}

	return nil
}

type CalculateTxSizeData struct {
	TxFullSize     decimal.Decimal
	TxStrippedSize decimal.Decimal
	Weight         decimal.Decimal
	VSize          decimal.Decimal
	TotalFee       decimal.Decimal
}

// CalculateTxSize calculates the size of the transaction and the fee for it.
//
// Docs: https://github.com/bitcoin/bips/blob/master/bip-0141.mediawiki#transaction-size-calculations
func (s *TxBuilder) CalculateTxSize(feePerByte decimal.Decimal) CalculateTxSizeData {
	txFullSize := decimal.NewFromInt(int64(s.tx.SerializeSize()))
	txStrippedSize := decimal.NewFromInt(int64(s.tx.SerializeSize()))

	weight := txStrippedSize.Mul(decimal.NewFromInt(3)).Add(txFullSize)
	vSize := weight.Div(decimal.NewFromInt(4))
	totalFee := vSize.Mul(feePerByte).Ceil()

	res := CalculateTxSizeData{
		TxFullSize:     txFullSize,
		TxStrippedSize: txStrippedSize,
		Weight:         weight,
		VSize:          vSize,
		TotalFee:       totalFee,
	}

	return res
}

// EmulateTxSize calculates the size of the transaction and the fee for it.
func (s *TxBuilder) EmulateTxSize(feePerByte decimal.Decimal) (CalculateTxSizeData, error) {
	// emulate transaction
	clonedTx1 := NewTransactionFromBuilder(s)

	if err := clonedTx1.SignTx(); err != nil {
		return CalculateTxSizeData{}, fmt.Errorf("sign cloned transaction 1: %w", err)
	}

	// calculate transaction size
	txSizeData := clonedTx1.CalculateTxSize(feePerByte)

	// emulate second transaction with the fee
	clonedTx2 := NewTransactionFromBuilder(s)

	// add output with the fee to the second transaction and sign it
	clonedTx2.MsgTx().TxOut[0].Value -= txSizeData.TotalFee.IntPart()
	if err := clonedTx2.SignTx(); err != nil {
		return CalculateTxSizeData{}, fmt.Errorf("sign cloned transaction 2: %w", err)
	}

	return clonedTx2.CalculateTxSize(feePerByte), nil
}
