package ltc

import (
	"encoding/hex"
	"fmt"

	"github.com/ltcsuite/ltcd/btcec/v2"
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/chaincfg/chainhash"
	"github.com/ltcsuite/ltcd/ltcutil"
	"github.com/ltcsuite/ltcd/txscript"
	"github.com/ltcsuite/ltcd/wire"
	"github.com/shopspring/decimal"
)

type TxInput struct {
	PrivateKey *btcec.PrivateKey
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

	txIn := wire.NewTxIn(outPoint, nil, nil)

	// Add RBF flag
	// More details: https://learnmeabitcoin.com/technical/transaction/input/sequence/#replace-by-fee-usage
	txIn.Sequence = 0xfffffffd
	s.tx.AddTxIn(txIn)
	s.Inputs = append(s.Inputs, input)

	return nil
}

func (s *TxBuilder) AddOutput(address string, amount decimal.Decimal) error {
	for _, out := range s.outputs {
		if out.address == address {
			return ErrOutputAlreadyExists
		}
	}

	addr, err := ltcutil.DecodeAddress(address, s.chainParams)
	if err != nil {
		return fmt.Errorf("decode address %s", address)
	}

	pk, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return fmt.Errorf("PayToAddrScript error: %w", err)
	}

	s.tx.AddTxOut(wire.NewTxOut(amount.IntPart(), pk))
	s.outputs = append(s.outputs, txOutput{address: address, amount: amount})

	return nil
}

// SignTx
func (s *TxBuilder) SignTx() error {
	if len(s.Inputs) == 0 {
		return fmt.Errorf("no inputs to sign")
	}

	multiFetcher := txscript.NewMultiPrevOutFetcher(nil)
	for i, input := range s.Inputs {
		scriptBytes, err := hex.DecodeString(input.PkScript)
		if err != nil {
			return fmt.Errorf("cannot decode pkScript for input %d: %w", i, err)
		}
		txHash, err := chainhash.NewHashFromStr(input.Hash)
		if err != nil {
			return err
		}

		outPoint := wire.NewOutPoint(txHash, input.Sequence)
		txOut := wire.NewTxOut(input.Amount, scriptBytes)

		multiFetcher.AddPrevOut(*outPoint, txOut)
	}

	// Calculate the signature hash for each input and sign it
	txSigHashes := txscript.NewTxSigHashes(s.tx, multiFetcher)

	for idx, input := range s.Inputs {
		if err := s.signTxIn(idx, input, txSigHashes, multiFetcher); err != nil {
			return fmt.Errorf("cannot sign input: %w", err)
		}
	}
	return nil
}

func (s *TxBuilder) signTxIn(index int, input TxInput, txSigHashes *txscript.TxSigHashes, outFetcher txscript.PrevOutputFetcher) error {
	scriptBytes, err := hex.DecodeString(input.PkScript)
	if err != nil {
		return fmt.Errorf("cannot decode source pk script: %w", err)
	}

	// P2PKH
	if txscript.IsPayToPubKeyHash(scriptBytes) {
		sig, err := txscript.SignatureScript(s.tx, index, scriptBytes, txscript.SigHashAll, input.PrivateKey, true)
		if err != nil {
			return fmt.Errorf("cannot create signature for P2PKH: %w", err)
		}
		s.tx.TxIn[index].SignatureScript = sig
	}

	// P2SH
	if txscript.IsPayToScriptHash(scriptBytes) {
		redeemScript, err := txscript.NewScriptBuilder().
			AddOp(txscript.OP_0).
			AddData(input.PrivateKey.PubKey().SerializeUncompressed()).
			Script()
		if err != nil {
			return fmt.Errorf("cannot create redeem script: %w", err)
		}

		sig, err := txscript.RawTxInSignature(s.tx, index, redeemScript, txscript.SigHashAll, input.PrivateKey)
		if err != nil {
			return fmt.Errorf("cannot create signature for P2SH: %w", err)
		}

		s.tx.TxIn[index].SignatureScript, err = txscript.NewScriptBuilder().
			AddOp(txscript.OP_0).
			AddData(sig).
			AddData(redeemScript).Script()
		if err != nil {
			return fmt.Errorf("cannot create signature script for P2SH: %w", err)
		}
	}

	// P2WPKH
	if txscript.IsPayToWitnessPubKeyHash(scriptBytes) {
		witness, err := txscript.WitnessSignature(s.tx, txSigHashes, index, input.Amount, scriptBytes, txscript.SigHashAll, input.PrivateKey, true)
		if err != nil {
			return fmt.Errorf("cannot create witness signature for P2WPKH: %w", err)
		}
		s.tx.TxIn[index].Witness = witness
	}

	// P2TR
	if txscript.IsPayToTaproot(scriptBytes) {
		s.tx.TxIn[index].Witness, err = txscript.TaprootWitnessSignature(s.tx, txSigHashes, index, input.Amount, scriptBytes, txscript.SigHashDefault, input.PrivateKey)
		if err != nil {
			return fmt.Errorf("cannot create Taproot witness signature: %w", err)
		}
	}

	flags := txscript.StandardVerifyFlags
	vm, err := txscript.NewEngine(scriptBytes, s.tx, index, flags, nil, txSigHashes, input.Amount, outFetcher)
	if err != nil {
		return fmt.Errorf("cannot create script engine: %w", err)
	}

	if err := vm.Execute(); err != nil {
		return fmt.Errorf("cannot execute vm script: %w", err)
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
	txStrippedSize := decimal.NewFromInt(int64(s.tx.SerializeSizeStripped()))

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
