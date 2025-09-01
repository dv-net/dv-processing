package doge

import (
	"fmt"

	"github.com/dv-net/go-bip39"
	"github.com/ltcsuite/ltcd/btcec/v2"
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/ltcutil"
	"github.com/ltcsuite/ltcd/ltcutil/hdkeychain"
)

type AddressType string

const (
	AddressTypeP2PKH AddressType = "P2PKH" // Legacy
	AddressTypeP2SH  AddressType = "P2SH"  // SegWit (nested)
)

const purpose = 44

func (t AddressType) Validate() error {
	switch t {
	case AddressTypeP2PKH, AddressTypeP2SH:
		return nil
	default:
		return fmt.Errorf("unsupported address type: %s", t)
	}
}

type WalletSDK struct {
	chainParams *chaincfg.Params
}

func NewWalletSDK(chainParams *chaincfg.Params) *WalletSDK {
	if chainParams == nil {
		chainParams = &DogecoinMainNetParams
	}

	return &WalletSDK{
		chainParams: chainParams,
	}
}

func (s *WalletSDK) ChainParams() *chaincfg.Params {
	return s.chainParams
}

type GenerateAddressData struct {
	chainParams *chaincfg.Params

	Address       ltcutil.Address
	PublicKey     *btcec.PublicKey
	PrivateKey    *btcec.PrivateKey
	PrivateKeyWIF *ltcutil.WIF
	MasterKey     *hdkeychain.ExtendedKey
	Sequence      uint32
}

func (s GenerateAddressData) AddressPubKey() (string, error) {
	address, err := ltcutil.NewAddressPubKey(s.PublicKey.SerializeCompressed(), s.chainParams)
	if err != nil {
		return "", fmt.Errorf("failed to create public key: %w", err)
	}

	return address.String(), nil
}

func (s *WalletSDK) GenerateAddress(mnemonic, passphrase string, sequenceNumber uint32) (*GenerateAddressData, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, passphrase)
	masterKey, err := hdkeychain.NewMaster(seed, s.chainParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	purposeKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + purpose)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose key: %w", err)
	}
	coinKey, err := purposeKey.Derive(hdkeychain.HardenedKeyStart + 3) // Dogecoin coin type
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

	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get EC private key: %w", err)
	}
	pubKey := privKey.PubKey()

	var addr ltcutil.Address

	pubKeyHash := ltcutil.Hash160(pubKey.SerializeCompressed())
	addr, err = ltcutil.NewAddressPubKeyHash(pubKeyHash, s.chainParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create P2PKH address: %w", err)
	}

	wif, err := ltcutil.NewWIF(privKey, s.chainParams, true)
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

func (s *WalletSDK) AddressFromPrivateKey(privateKeyWIF string) (string, *btcec.PrivateKey, error) {
	wif, err := ltcutil.DecodeWIF(privateKeyWIF)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode WIF: %w", err)
	}
	privKey := wif.PrivKey
	pubKey := privKey.PubKey()

	pubKeyHash := ltcutil.Hash160(pubKey.SerializeCompressed())
	addr, err := ltcutil.NewAddressPubKeyHash(pubKeyHash, s.chainParams)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return addr.EncodeAddress(), privKey, nil
}

func (s *WalletSDK) DecodeAddressType(address string) (AddressType, error) {
	return DecodeAddressType(address, s.chainParams)
}

func (s *WalletSDK) ValidateAddress(address string) bool {
	_, err := ltcutil.DecodeAddress(address, s.chainParams)
	return err == nil
}

func DecodeAddressType(address string, chainParams *chaincfg.Params) (AddressType, error) {
	addr, err := ltcutil.DecodeAddress(address, chainParams)
	if err != nil {
		return "", fmt.Errorf("failed to decode address: %w", err)
	}

	switch addr := addr.(type) {
	case *ltcutil.AddressPubKeyHash:
		return AddressTypeP2PKH, nil
	case *ltcutil.AddressScriptHash:
		return AddressTypeP2SH, nil
	default:
		return "", fmt.Errorf("unknown address type: %T", addr)
	}
}
