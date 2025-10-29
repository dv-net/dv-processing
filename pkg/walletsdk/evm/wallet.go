package evm

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/dv-net/go-bip39"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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
		"44'/60'/0'/0/"+strconv.Itoa(int(sequence)),
	)
	if err != nil {
		return "", nil, nil, errors.New("failed to derive private key")
	}

	privateKey, err := crypto.ToECDSA(secret[:])
	if err != nil {
		return "", nil, nil, errors.New("failed to generate ECDSA from secret")
	}
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	return strings.ToLower(address.String()), privateKey, &privateKey.PublicKey, nil
}

func AddressWallet(mnemonic string, passphrase string, sequence uint32) (string, error) {
	address, _, _, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}

	return strings.ToLower(address), nil
}

func AddressSecret(address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	wAddress, private, _, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(address, wAddress) {
		return "", errors.New("generate private key address")
	}
	return hexutil.Encode(crypto.FromECDSA(private)), nil
}

func AddressPublic(address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	wAddress, _, public, err := WalletPubKeyHash(mnemonic, passphrase, sequence)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(address, wAddress) {
		return "", errors.New("generate private key address")
	}
	return hex.EncodeToString(crypto.FromECDSAPub(public)), nil
}

func (s WalletSDK) ValidateAddress(address string) bool {
	return ValidateAddress(address)
}
