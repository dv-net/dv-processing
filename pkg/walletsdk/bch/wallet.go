package bch

import (
	"fmt"
	"strings"

	"github.com/dv-net/go-bip39"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	"github.com/gcash/bchutil/base58"
	"github.com/gcash/bchutil/hdkeychain"
)

type WalletSDK struct {
	chainParams *chaincfg.Params
}

// NewWalletSDK creates a new WalletSDK instance.
//
// If chainParams is nil, the mainnet parameters will be used.
func NewWalletSDK(chainParams *chaincfg.Params) *WalletSDK {
	if chainParams == nil {
		chainParams = &chaincfg.MainNetParams
	}

	return &WalletSDK{
		chainParams: chainParams,
	}
}

// ChainParams returns the chain parameters for the wallet
func (s WalletSDK) ChainParams() *chaincfg.Params {
	return s.chainParams
}

type GenerateAddressData struct {
	chainParams *chaincfg.Params

	Address       *bchutil.AddressPubKeyHash
	PublicKey     *bchec.PublicKey
	PrivateKey    *bchec.PrivateKey
	PrivateKeyWIF *bchutil.WIF
	MasterKey     *hdkeychain.ExtendedKey
	Sequence      uint32
}

func (s GenerateAddressData) AddressPubKey() (string, error) {
	address, err := bchutil.NewAddressPubKey(s.PublicKey.SerializeCompressed(), s.chainParams)
	if err != nil {
		return "", fmt.Errorf("failed to create public key: %w", err)
	}

	return address.String(), nil
}

func (s WalletSDK) GenerateAddress(mnemonic, passphrase string, sequenceNumber uint32) (*GenerateAddressData, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, passphrase)

	masterKey, err := hdkeychain.NewMaster(seed, s.chainParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Derive the purpose key (44' for BIP-44)
	purposeKey, err := masterKey.Child(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose key (44'): %w", err)
	}

	// Derive the coin type key (145' for BCH)
	coinTypeKey, err := purposeKey.Child(hdkeychain.HardenedKeyStart + s.chainParams.HDCoinType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin type key (145'): %w", err)
	}

	// Derive the account key (0' for the first account)
	accountKey, err := coinTypeKey.Child(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account key (0'): %w", err)
	}

	// Derive the change key (0 for external addresses)
	changeKey, err := accountKey.Child(0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change key: %w", err)
	}

	// Derive the address key using the sequence number
	addrKey, err := changeKey.Child(sequenceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address key: %w", err)
	}

	// Generate the public key from the address key
	pubKey, err := addrKey.ECPubKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	privKey, err := addrKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	privateKeyWIF, err := bchutil.NewWIF(privKey, s.chainParams, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create private WIF: %w", err)
	}

	// Create HASH160 of the public key
	pubKeyHash := bchutil.Hash160(pubKey.SerializeCompressed())

	address, err := bchutil.NewAddressPubKeyHash(pubKeyHash, s.chainParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create public address: %w", err)
	}

	return &GenerateAddressData{
		chainParams: s.chainParams,

		Address:       address,
		PublicKey:     pubKey,
		PrivateKey:    privKey,
		PrivateKeyWIF: privateKeyWIF,
		MasterKey:     masterKey,
		Sequence:      sequenceNumber,
	}, nil
}

func (s WalletSDK) AddressFromPrivateKey(privateKeyWIF string) (string, *bchec.PrivateKey, error) {
	wif, err := bchutil.DecodeWIF(privateKeyWIF)
	if err != nil {
		return "", nil, fmt.Errorf("decode WIF: %w", err)
	}

	pubKeyHash := bchutil.Hash160(wif.PrivKey.PubKey().SerializeCompressed())

	address, err := bchutil.NewAddressPubKeyHash(pubKeyHash, s.chainParams)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create public address: %w", err)
	}

	return address.String(), wif.PrivKey, nil
}

func DecodeAddressToCashAddr(address string, params *chaincfg.Params) (string, error) {
	addr, err := bchutil.DecodeAddress(address, params)
	if err != nil {
		return "", fmt.Errorf("failed to decode address: %w", err)
	}

	if strings.HasPrefix(address, "1") || strings.HasPrefix(address, "3") || strings.HasPrefix(address, "2") {
		switch a := addr.(type) {
		// LegacyAddressPubKeyHash - legacy-address P2PKH
		case *bchutil.LegacyAddressPubKeyHash:
			cashAddr, err := bchutil.NewAddressPubKeyHash(a.Hash160()[:], params)
			if err != nil {
				return "", err
			}

			return cashAddr.String(), nil
		// LegacyAddressScriptHash - legacy-address P2SH
		case *bchutil.LegacyAddressScriptHash:
			cashAddr, err := bchutil.NewAddressScriptHashFromHash(a.Hash160()[:], params)
			if err != nil {
				return "", err
			}

			return cashAddr.String(), nil
		default:
			return "", fmt.Errorf("unsupported address type %T", addr)
		}
	}

	return addr.String(), nil
}

func DecodeAddressToLegacyAddr(address string, params *chaincfg.Params) (string, error) {
	addr, err := bchutil.DecodeAddress(address, params)
	if err != nil {
		return "", fmt.Errorf("failed to decode address: %w", err)
	}

	switch a := addr.(type) {
	// If the address is already Legacy, we return its string performance.
	case *bchutil.LegacyAddressPubKeyHash:
		return a.String(), nil
	case *bchutil.LegacyAddressScriptHash:
		return a.String(), nil

	// If the address is set in cashddr format, it is necessary to create Legacy.
	case *bchutil.AddressPubKeyHash:
		// We transform into Legacy -format using Base58 - Checkencode.
		legacy := base58.CheckEncode(a.Hash160()[:], params.LegacyPubKeyHashAddrID)
		return legacy, nil
	case *bchutil.AddressScriptHash:
		legacy := base58.CheckEncode(a.Hash160()[:], params.LegacyScriptHashAddrID)
		return legacy, nil
	default:
		return "", fmt.Errorf("unsupported address type %T", addr)
	}
}

func IsCashAddrAddress(address string, params *chaincfg.Params) (bool, error) {
	addr, err := bchutil.DecodeAddress(address, params)
	if err != nil {
		return false, fmt.Errorf("failed to decode address: %w", err)
	}

	switch addr.(type) {
	// AddressPubKeyHash - cashAddr-address P2PKH
	case *bchutil.AddressPubKeyHash:
		return true, nil
	// AddressScriptHash - cashAddr-address P2SH
	case *bchutil.AddressScriptHash:
		return true, nil
	default:
		return false, nil
	}
}

func IsLegacyAddress(address string, params *chaincfg.Params) (bool, error) {
	addr, err := bchutil.DecodeAddress(address, params)
	if err != nil {
		return false, fmt.Errorf("failed to decode address: %w", err)
	}

	switch addr.(type) {
	// LegacyAddressPubKeyHash - legacy-address P2PKH
	case *bchutil.LegacyAddressPubKeyHash:
		return true, nil
	// LegacyAddressScriptHash - legacy-address P2SH
	case *bchutil.LegacyAddressScriptHash:
		return true, nil
	default:
		return false, nil
	}
}

// Bitcoin Cash
func (s WalletSDK) ValidateAddress(address string) bool {
	_, err := bchutil.DecodeAddress(address, s.chainParams)
	return err == nil
}
