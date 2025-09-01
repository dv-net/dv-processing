package doge_test

import (
	"fmt"
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/dv-net/go-bip39"
)

func TestGenerateAddress_AllTypes(t *testing.T) {
	// Valid mnemonic and passphrase
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "testpass"
	sdk := doge.NewWalletSDK(&doge.DogecoinMainNetParams)

	// Test cases for all supported address types
	testCases := []struct {
		addrType doge.AddressType
		seq      uint32
	}{
		{doge.AddressTypeP2PKH, 0},
	}

	for _, tc := range testCases {
		t.Run(string(tc.addrType), func(t *testing.T) {
			data, err := sdk.GenerateAddress(mnemonic, passphrase, tc.seq)
			if err != nil {
				t.Errorf("failed to generate address for type %s: %v", tc.addrType, err)
				return
			}

			// Check address string
			addrStr := data.Address.EncodeAddress()
			if addrStr == "" {
				t.Errorf("generated address is empty for type %s", tc.addrType)
			}

			// Verify decoded address type
			decodedType, err := sdk.DecodeAddressType(addrStr)
			if err != nil {
				t.Errorf("failed to decode address type for %s: %v", addrStr, err)
				return
			}
			if decodedType != tc.addrType {
				t.Errorf("decoded type mismatch for address %s: expected %s, got %s", addrStr, tc.addrType, decodedType)
			}

			// Check public key
			pubKeyStr, err := data.AddressPubKey()
			if err != nil {
				t.Errorf("AddressPublicKey returned error for type %s: %v", tc.addrType, err)
			}
			if pubKeyStr == "" {
				t.Errorf("generated public key string is empty for type %s", tc.addrType)
			}

			fmt.Printf("Address: %s, PubKey: %s, PrivKeyWIF: %s\n", addrStr, pubKeyStr, data.PrivateKeyWIF.String())
		})
	}
}

func TestAddressFromPrivateKey_P2PKH(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "testpass"
	sdk := doge.NewWalletSDK(&doge.DogecoinMainNetParams)

	data, err := sdk.GenerateAddress(mnemonic, passphrase, 4)
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
	mnemonic := "inform under analyst dynamic upset term identify play zone praise buffalo please"
	passphrase := "testpass"
	sdk := doge.NewWalletSDK(&doge.DogecoinMainNetParams)

	testCases := []struct {
		addrType doge.AddressType
		seq      uint32
	}{
		{doge.AddressTypeP2PKH, 0},
	}

	for _, tc := range testCases {
		t.Run(string(tc.addrType), func(t *testing.T) {
			data, err := sdk.GenerateAddress(mnemonic, passphrase, tc.seq)
			if err != nil {
				t.Errorf("GenerateAddress returned error for type %s: %v", tc.addrType, err)
				return
			}
			addrStr := data.Address.EncodeAddress()
			decodedType, err := sdk.DecodeAddressType(addrStr)
			if err != nil {
				t.Errorf("DecodeAddressType returned error for address %s: %v", addrStr, err)
				return
			}
			if decodedType != tc.addrType {
				t.Errorf("expected address type %s, got %s for address %s", tc.addrType, decodedType, addrStr)
			}
		})
	}
}

func TestInvalidMnemonic(t *testing.T) {
	invalidMnemonic := "invalid mnemonic phrase"
	passphrase := "testpass"
	sdk := doge.NewWalletSDK(&doge.DogecoinMainNetParams)

	_, err := sdk.GenerateAddress(invalidMnemonic, passphrase, 0)
	if err == nil {
		t.Fatal("expected error for invalid mnemonic, got nil")
	}
}

func TestEmptyPassphrase(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := ""
	sdk := doge.NewWalletSDK(&doge.DogecoinMainNetParams)

	data, err := sdk.GenerateAddress(mnemonic, passphrase, 0)
	if err != nil {
		t.Errorf("unexpected error for empty passphrase: %v", err)
	}
	if data == nil {
		t.Fatal("expected valid address data for empty passphrase")
	}
}

func TestBip39Consistency(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if !bip39.IsMnemonicValid(mnemonic) {
		t.Fatal("expected mnemonic to be valid")
	}
}

func TestValidateAddress(t *testing.T) {
	sdk := doge.NewWalletSDK(&doge.DogecoinMainNetParams)

	testCases := []struct {
		name     string
		address  string
		expected bool
	}{
		{
			name:     "Valid P2PKH Mainnet",
			address:  "DEgDVFa2DoW1533dxeDVdTxQFhMzs1pMke", // Example Dogecoin P2PKH address
			expected: true,
		},
		{
			name:     "Valid P2SH Mainnet",
			address:  "9srEbLELgnH8rQ69Mcb35es2p68aeG9fZw", // Example Dogecoin P2SH address
			expected: true,
		},
		{
			name:     "Valid P2SH Mainnet",
			address:  "A4VHXaodZ629J6WqLgCcpMDjBHzpzrKeDx", // Example Dogecoin P2SH address
			expected: true,
		},
		{
			name:     "Invalid Address",
			address:  "invalidaddress123",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sdk.ValidateAddress(tc.address)
			if result != tc.expected {
				t.Errorf("ValidateAddress(%s) = %v; expected %v", tc.address, result, tc.expected)
			}
		})
	}
}
