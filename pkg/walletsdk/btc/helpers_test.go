package btc_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
)

const (
	mnemonic   = "bubble edit online huge reveal forest mirror tongue glance dish august group machine hello equal"
	passphrase = "asfdasdfasdfasdf" //nolint:gosec
)

func generateTestAddresses(count int, addrType btc.AddressType) ([]*btc.GenerateAddressData, error) {
	wsdk := btc.NewWalletSDK(&chaincfg.MainNetParams)

	var addresses []*btc.GenerateAddressData
	for i := range count {
		addrData, err := wsdk.GenerateAddress(addrType, mnemonic, passphrase, uint32(i))
		if err != nil {
			return nil, fmt.Errorf("generate test segwit addresses: %w", err)
		}
		addresses = append(addresses, addrData)
	}

	return addresses, nil
}

func generateRandomTxHash() (string, error) {
	var hashBytes [chainhash.HashSize]byte
	// We fill the array with random bytes
	if _, err := rand.Read(hashBytes[:]); err != nil {
		return "", fmt.Errorf("error generating random bytes: %v", err)
	}
	// We convert an array into type chainhash.hash
	txHash := chainhash.Hash(hashBytes)
	return txHash.String(), nil
}

func generateRandomSatoshiAmount(min, max int64) int64 {
	randAmount, _ := rand.Int(rand.Reader, big.NewInt(max-min+1))
	return randAmount.Int64() + min
}

func generateSegwitPkScript(address string) (string, error) {
	addressHash := chainhash.HashB([]byte(address))
	if len(addressHash) < 20 {
		return "", fmt.Errorf("address hash is too short for pk script")
	}

	// OP_0 <20-byte hash>
	pkScript := append([]byte{0x00, 0x14}, addressHash[:20]...)
	return hex.EncodeToString(pkScript), nil
}

// func generateRandomSequence() int32 {
// 	maxSequence := int32(0xFFFFFF)
// 	randSeq, _ := rand.Int(rand.Reader, big.NewInt(int64(maxSequence)))
// 	return int32(randSeq.Int64())
// }

func generateRandomUTXOForAddress(address *btc.GenerateAddressData, numUTXOs uint) ([]btc.TxInput, error) {
	if numUTXOs == 0 {
		return nil, fmt.Errorf("numUTXOs must be greater than 0")
	}

	var utxos []btc.TxInput
	for range numUTXOs {
		// Generation of random txhash (64-symbol hash)
		txHash, err := generateRandomTxHash()
		if err != nil {
			return nil, fmt.Errorf("failed to generate tx hash: %w", err)
		}

		// Generation of a random amount (in Satoshi)
		amount := generateRandomSatoshiAmount(1, 10000000) // From 1 to 10 million satoshi

		// Generation of a random PkScript (Hash address Segwit P2WPKH)
		pkScript, err := generateSegwitPkScript(address.Address.String())
		if err != nil {
			return nil, fmt.Errorf("failed to generate pk script: %w", err)
		}

		utxos = append(utxos, btc.TxInput{
			PrivateKey: address.PrivateKey,
			PkScript:   pkScript,
			Hash:       txHash,
			Sequence:   address.Sequence,
			Amount:     amount,
		})
	}

	return utxos, nil
}
