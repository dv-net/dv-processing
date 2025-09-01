package btc

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/dv-net/go-bip39"
)

// AddressType represents the type of a Bitcoin address.
type AddressType string

const (
	AddressTypeP2PKH  AddressType = "P2PKH"  // Legacy
	AddressTypeP2SH   AddressType = "P2SH"   // SegWit (nested)
	AddressTypeP2WPKH AddressType = "P2WPKH" // Native SegWit (Bech32)
	AddressTypeP2TR   AddressType = "P2TR"   // Taproot
)

// Validate
func (t AddressType) Validate() error {
	switch t {
	case AddressTypeP2PKH, AddressTypeP2SH, AddressTypeP2WPKH, AddressTypeP2TR:
		return nil
	default:
		return fmt.Errorf("unsupported address type: %s", t)
	}
}

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

	Address       btcutil.Address
	PublicKey     *btcec.PublicKey
	PrivateKey    *btcec.PrivateKey
	PrivateKeyWIF *btcutil.WIF
	MasterKey     *hdkeychain.ExtendedKey
	Sequence      uint32
}

func (s GenerateAddressData) AddressPubKey() (string, error) {
	address, err := btcutil.NewAddressPubKey(s.PublicKey.SerializeCompressed(), s.chainParams)
	if err != nil {
		return "", fmt.Errorf("failed to create public key: %w", err)
	}

	return address.String(), nil
}

func (s WalletSDK) GenerateAddress(addressType AddressType, mnemonic, passphrase string, sequenceNumber uint32) (*GenerateAddressData, error) {
	// Check mnemonic and passphrase
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	if passphrase == "" {
		return nil, fmt.Errorf("passphrase is required")
	}

	if err := addressType.Validate(); err != nil {
		return nil, err
	}

	// get seed from mnemonic and passphrase
	seed := bip39.NewSeed(mnemonic, passphrase)

	// Create master key
	masterKey, err := hdkeychain.NewMaster(seed, s.chainParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Get purpose key based on address type
	var purpose uint32
	switch addressType {
	case AddressTypeP2PKH:
		purpose = 44
	case AddressTypeP2SH:
		purpose = 49
	case AddressTypeP2WPKH:
		purpose = 84
	case AddressTypeP2TR:
		purpose = 86
	default:
		return nil, fmt.Errorf("unsupported address type")
	}

	// Derivation path: m / purpose' / 0' / 0' / 0 / sequenceNumber.
	purposeKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + purpose)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose key: %w", err)
	}
	coinKey, err := purposeKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin key: %w", err)
	}
	accountKey, err := coinKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account key: %w", err)
	}
	changeKey, err := accountKey.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change key: %w", err)
	}
	childKey, err := changeKey.Derive(sequenceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address key: %w", err)
	}

	// Get private key and public keys
	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get EC private key: %w", err)
	}
	pubKey := privKey.PubKey()

	var addr btcutil.Address

	switch addressType {
	case AddressTypeP2PKH: // Legacy address P2PKH.
		pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
		addr, err = btcutil.NewAddressPubKeyHash(pubKeyHash, s.chainParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create P2PKH address: %w", err)
		}
	case AddressTypeP2SH: // SegWit (nested segwit) address
		pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
		witAddr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, s.chainParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create witness address: %w", err)
		}
		redeemScript, err := txscript.PayToAddrScript(witAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to create redeem script: %w", err)
		}
		addr, err = btcutil.NewAddressScriptHash(redeemScript, s.chainParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create P2SH address: %w", err)
		}
	case AddressTypeP2WPKH: // Native SegWit (Bech32) address
		pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
		addr, err = btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, s.chainParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create P2WPKH address: %w", err)
		}
	case AddressTypeP2TR: // Taproot address
		tapKey := txscript.ComputeTaprootKeyNoScript(pubKey)
		tweakedPubKeyHash := schnorr.SerializePubKey(tapKey)
		addr, err = btcutil.NewAddressTaproot(tweakedPubKeyHash, s.chainParams)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported address type")
	}

	// Generate WIF from private key.
	wif, err := btcutil.NewWIF(privKey, s.chainParams, true)
	if err != nil {
		return nil, fmt.Errorf("failed to generate WIF: %w", err)
	}

	data := &GenerateAddressData{
		chainParams:   s.chainParams,
		Address:       addr,
		PublicKey:     pubKey,
		PrivateKey:    privKey,
		PrivateKeyWIF: wif,
		MasterKey:     masterKey,
		Sequence:      sequenceNumber,
	}

	return data, nil
}

func (s WalletSDK) AddressFromPrivateKey(privateKeyWIF string) (string, *btcec.PrivateKey, error) {
	wif, err := btcutil.DecodeWIF(privateKeyWIF)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode WIF: %w", err)
	}
	privKey := wif.PrivKey
	pubKey := privKey.PubKey()

	pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, s.chainParams)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return addr.EncodeAddress(), privKey, nil
}

func (s WalletSDK) DecodeAddressType(address string) (AddressType, error) {
	addr, err := btcutil.DecodeAddress(address, s.chainParams)
	if err != nil {
		return "", fmt.Errorf("failed to decode address: %w", err)
	}

	switch addr := addr.(type) {
	case *btcutil.AddressPubKeyHash:
		return AddressTypeP2PKH, nil
	case *btcutil.AddressScriptHash:
		return AddressTypeP2SH, nil
	case *btcutil.AddressWitnessPubKeyHash:
		return AddressTypeP2WPKH, nil
	case *btcutil.AddressTaproot:
		return AddressTypeP2TR, nil
	default:
		return "", fmt.Errorf("unknown address type: %T", addr)
	}
}

func (s WalletSDK) ValidateAddress(address string) bool {
	_, err := btcutil.DecodeAddress(address, s.chainParams)
	return err == nil
}
