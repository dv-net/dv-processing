package tron

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strconv"

	"github.com/dv-net/go-bip39"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	addr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
)

type WalletSDK struct{}

// NewWalletSDK creates a new WalletSDK instance.
func NewWalletSDK() *WalletSDK {
	return &WalletSDK{}
}

func WalletPubKeyHash(mnemonic string, passphrase string, sequence uint32) (string, *ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	seed := bip39.NewSeed(mnemonic, passphrase)

	secret, chainCode := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	secret, err := hd.DerivePrivateKeyForPath(
		crypto.S256(),
		secret,
		chainCode,
		"44'/195'/0'/0/"+strconv.Itoa(int(sequence)),
	)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to derive private key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(secret[:])
	if err != nil {
		return "", nil, nil, errors.New("failed to generate ECDSA from secret")
	}

	address := addr.PubkeyToAddress(privateKey.PublicKey)

	return address.String(), privateKey, &privateKey.PublicKey, nil
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
	return hexutil.Encode(crypto.FromECDSA(private)), nil
}

func AddressPublic(address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	wAddress, _, public, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}
	if address != wAddress {
		return "", fmt.Errorf("generate private key address")
	}
	return hexutil.Encode(crypto.FromECDSAPub(public)), nil
}

func WalletFromPrivateKeyBytes(privateKey []byte) (string, *ecdsa.PrivateKey, *ecdsa.PublicKey) {
	private, err := crypto.ToECDSA(privateKey)
	if err != nil {
		return "", nil, nil
	}
	address := addr.PubkeyToAddress(private.PublicKey)
	return address.String(), private, &private.PublicKey
}

func (s WalletSDK) ValidateAddress(address string) bool {
	return ValidateAddress(address)
}
