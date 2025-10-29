package evm_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/go-bip39"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
	"github.com/stretchr/testify/require"
)

const (
	mnemonic        = "vague wool express sniff alley core hen symptom end rather month cave cross elder nest bright paddle use voice wife dolphin mosquito inside curve"
	passphrase      = "" //nolint:gosec
	defaultSequence = 0
)

func Test_GenerateAddresses(t *testing.T) {
	for i := range 5 {
		addr, priv, _, err := evm.WalletPubKeyHash(mnemonic, passphrase, uint32(i))
		if err != nil {
			t.Fatalf("failed to generate addresss: %v", err)
		}

		fmt.Println(addr, hexutil.Encode(crypto.FromECDSA(priv)))
	}
}

func TestEthereumWalletPubKeyHash(t *testing.T) {
	seed := bip39.NewSeed(mnemonic, passphrase)
	require.NotEmpty(t, seed)
	secret, chainCode := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	require.NotEmpty(t, secret)
	require.NotEmpty(t, chainCode)
	secret, err := hd.DerivePrivateKeyForPath(
		crypto.S256(),
		secret,
		chainCode,
		"44'/60'/0'/0/"+strconv.Itoa(defaultSequence),
	)
	require.NoError(t, err)
	require.NotEmpty(t, secret)

	privateKey, err := crypto.ToECDSA(secret[:])
	require.NoError(t, err)
	require.NotEmpty(t, privateKey)

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	require.NotEmpty(t, address)

	t.Log(address.String())
	t.Log(hexutil.Encode(crypto.FromECDSA(privateKey)))
	t.Log(hexutil.Encode(crypto.FromECDSAPub(&privateKey.PublicKey)))
}
