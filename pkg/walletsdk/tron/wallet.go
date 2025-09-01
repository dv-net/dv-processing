package tron

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/dv-net/go-bip39"
	addr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
)

type WalletSDK struct{}

// NewWalletSDK creates a new WalletSDK instance.
func NewWalletSDK() *WalletSDK {
	return &WalletSDK{}
}

func WalletPubKeyHash(mnemonic string, passphrase string, sequence uint32) (string, *btcec.PrivateKey, *btcec.PublicKey, error) {
	seed := bip39.NewSeed(mnemonic, passphrase)

	secret, chainCode := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	secret, err := hd.DerivePrivateKeyForPath(
		btcec.S256(),
		secret,
		chainCode,
		"44'/195'/0'/0/"+strconv.Itoa(int(sequence)),
	)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to derive private key: %w", err)
	}

	privateKey, publicKey := btcec.PrivKeyFromBytes(secret[:])

	address := addr.PubkeyToAddress(*publicKey.ToECDSA()).String()

	return address, privateKey, publicKey, nil
}

func AddressWallet(mnemonic string, passphrase string, sequence uint32) (string, error) {
	address, _, _, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}

	return address, nil
}

func AddressSecret(address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	wAddress, private, _, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}
	if address != wAddress {
		return "", fmt.Errorf("generate private key address")
	}
	return private.Key.String(), nil
}

func AddressPublic(address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	wAddress, _, public, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}
	if address != wAddress {
		return "", fmt.Errorf("generate private key address")
	}
	return hex.EncodeToString(public.SerializeCompressed()), nil
}

func WalletFromPrivateKeyBytes(privateKey []byte) (string, *btcec.PrivateKey, *btcec.PublicKey) {
	priv, pub := btcec.PrivKeyFromBytes(privateKey)
	return addr.PubkeyToAddress(*pub.ToECDSA()).String(), priv, pub
}

func (s WalletSDK) ValidateAddress(address string) bool {
	return ValidateAddress(address)
}
