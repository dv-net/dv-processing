package ltc_test

import (
	"fmt"
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/go-bip39"
	"github.com/ltcsuite/ltcd/chaincfg"
)

func TestGenerateAddress_AllTypes(t *testing.T) {
	// Valid mnemonics and Passphrase.
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "testpass"
	sdk := ltc.NewWalletSDK(&chaincfg.MainNetParams)

	// Test cases for all types of addresses.
	testCases := []struct {
		addrType ltc.AddressType
		seq      uint32
	}{
		{ltc.AddressTypeP2PKH, 0},
		{ltc.AddressTypeP2SH, 1},
		{ltc.AddressTypeP2WPKH, 2},
		{ltc.AddressTypeP2TR, 3},
	}

	for _, tc := range testCases {
		data, err := sdk.GenerateAddress(tc.addrType, mnemonic, passphrase, tc.seq)
		if err != nil {
			t.Errorf("failed to generate address for type %s: %v", tc.addrType, err)
			continue
		}
		// We get a stringed presentation of the address through the Address field.
		addrStr := data.Address.EncodeAddress()
		if addrStr == "" {
			t.Errorf("generated address is empty for type %s", tc.addrType)
		}

		// We check that DecodeAddressType returns the expected type.
		decodedType, err := sdk.DecodeAddressType(addrStr)
		if err != nil {
			t.Errorf("failed to decode address type for %s: %v", addrStr, err)
			continue
		}
		if decodedType != tc.addrType {
			t.Errorf("decoded type mismatch for address %s: expected %s, got %s",
				addrStr, tc.addrType, decodedType)
		}

		// We check that the Addresspubkey method returns a non -why public key.
		pubKeyStr, err := data.AddressPubKey()
		if err != nil {
			t.Errorf("AddressPubKey returned error for type %s: %v", tc.addrType, err)
		}
		if pubKeyStr == "" {
			t.Errorf("generated public key string is empty for type %s", tc.addrType)
		}

		fmt.Println(addrStr, pubKeyStr, data.PrivateKeyWIF.String())
	}
}

func TestAddressFromPrivateKey_P2PKH(t *testing.T) {
	// We test the receipt of the address from the private key (P2PKH).
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "testpass"
	sdk := ltc.NewWalletSDK(&chaincfg.MainNetParams)

	data, err := sdk.GenerateAddress(ltc.AddressTypeP2PKH, mnemonic, passphrase, 4)
	if err != nil {
		t.Fatalf("GenerateAddress returned error: %v", err)
	}

	addrStr1 := data.Address.EncodeAddress()

	wifStr := data.PrivateKeyWIF.String()
	addrStr2, privKey, err := sdk.AddressFromPrivateKey(wifStr)
	if err != nil {
		t.Fatalf("AddressFromPrivateKey returned error: %v", err)
	}

	if addrStr1 != addrStr2 {
		t.Errorf("addresses do not match: generated=%s, fromWIF=%s", addrStr1, addrStr2)
	}
	if privKey == nil {
		t.Fatal("returned private key is nil")
	}
}

func TestDecodeAddressType_AllTypes(t *testing.T) {
	// We generate addresses for all types and check the correctness of the type determination.
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "testpass"
	sdk := ltc.NewWalletSDK(&chaincfg.MainNetParams)

	testCases := []struct {
		addrType ltc.AddressType
		seq      uint32
	}{
		{ltc.AddressTypeP2PKH, 5},
		{ltc.AddressTypeP2SH, 6},
		{ltc.AddressTypeP2WPKH, 7},
		{ltc.AddressTypeP2TR, 8},
	}

	for _, tc := range testCases {
		data, err := sdk.GenerateAddress(tc.addrType, mnemonic, passphrase, tc.seq)
		if err != nil {
			t.Errorf("GenerateAddress returned error for type %s: %v", tc.addrType, err)
			continue
		}
		addrStr := data.Address.EncodeAddress()
		decodedType, err := sdk.DecodeAddressType(addrStr)
		if err != nil {
			t.Errorf("DecodeAddressType returned error for address %s: %v", addrStr, err)
			continue
		}
		if decodedType != tc.addrType {
			t.Errorf("expected address type %s, got %s for address %s", tc.addrType, decodedType, addrStr)
		}
	}
}

func TestInvalidMnemonic(t *testing.T) {
	invalidMnemonic := "invalid mnemonic phrase"
	passphrase := "testpass"
	sdk := ltc.NewWalletSDK(&chaincfg.MainNetParams)

	_, err := sdk.GenerateAddress(ltc.AddressTypeP2PKH, invalidMnemonic, passphrase, 0)
	if err == nil {
		t.Fatal("expected error for invalid mnemonic, got nil")
	}
}

func TestEmptyPassphrase(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := ""
	sdk := ltc.NewWalletSDK(&chaincfg.MainNetParams)

	_, err := sdk.GenerateAddress(ltc.AddressTypeP2PKH, mnemonic, passphrase, 0)
	if err == nil {
		t.Fatal("expected error for empty passphrase, got nil")
	}
}

func TestBip39Consistency(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if !bip39.IsMnemonicValid(mnemonic) {
		t.Fatal("expected mnemonic to be valid")
	}
}
