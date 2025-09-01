package evm_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/go-bip39"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
	"github.com/stretchr/testify/require"
)

const (
	mnemonic        = "vague wool express sniff alley core hen symptom end rather month cave cross elder nest bright paddle use voice wife dolphin mosquito inside curve"
	passphrase      = "asdfasdfasdf" //nolint:gosec
	defaultSequence = 1
)

func Test_GenerateAddresses(t *testing.T) {
	for i := range 5 {
		addr, priv, _, err := evm.WalletPubKeyHash(mnemonic, passphrase, uint32(i))
		if err != nil {
			t.Fatalf("failed to generate addresss: %v", err)
		}

		fmt.Println(addr, priv)
	}
}

func TestEthereumWalletPubKeyHash(t *testing.T) {
	seed := bip39.NewSeed(mnemonic, passphrase)
	require.NotEmpty(t, seed)
	secret, chainCode := hd.ComputeMastersFromSeed(seed, []byte(passphrase))
	require.NotEmpty(t, secret)
	require.NotEmpty(t, chainCode)
	secret, err := hd.DerivePrivateKeyForPath(
		btcec.S256(),
		secret,
		chainCode,
		"44'/60'/0'/0/"+strconv.Itoa(defaultSequence),
	)
	require.NoError(t, err)
	require.NotEmpty(t, secret)

	privateKey, publicKey := secp.PrivKeyFromBytes(secret[:]), secp.PrivKeyFromBytes(secret[:]).PubKey()
	require.NotEmpty(t, privateKey)
	require.NotEmpty(t, publicKey)

	address := crypto.PubkeyToAddress(*publicKey.ToECDSA())
	require.NotEmpty(t, address)
	t.Log(address.String())

	t.Log(publicKey.ToECDSA())
	t.Log(privateKey.Key.String())
}
