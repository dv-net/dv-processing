package tron_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/dv-net/dv-processing/pkg/testutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/go-bip39"
	"github.com/ethereum/go-ethereum/common/hexutil"
	addr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/stretchr/testify/require"
)

const (
	mnemonic        = "vague wool express sniff alley core hen symptom end rather month cave cross elder nest bright paddle use voice wife dolphin mosquito inside curve"
	passphrase      = "" //nolint:gosec
	defaultSequence = 1
)

func Test_GenerateAddresses(t *testing.T) {
	for i := range 5 {
		addr, _, _, err := tron.WalletPubKeyHash(mnemonic, passphrase, uint32(i))
		if err != nil {
			t.Fatalf("failed to generate addresss: %v", err)
		}

		fmt.Println(addr)
	}
}

// TestTronWalletPubKeyHash - validates that we indeed generate the same address,private/public key for mnemonic.
// passphrase omitted for speed
func TestTronWalletPubKeyHash(t *testing.T) {
	t.Run("", func(t *testing.T) {
		seed := bip39.NewSeed(mnemonic, passphrase)
		require.NotEmpty(t, seed)
		secret, chainCode := hd.ComputeMastersFromSeed(seed, []byte(passphrase))
		require.NotEmpty(t, secret)
		require.NotEmpty(t, chainCode)
		secret, err := hd.DerivePrivateKeyForPath(
			btcec.S256(),
			secret,
			chainCode,
			"44'/195'/0'/0/"+strconv.Itoa(defaultSequence),
		)
		require.NoError(t, err)
		require.NotEmpty(t, secret)

		privateKey, publicKey := secp.PrivKeyFromBytes(secret[:]), secp.PrivKeyFromBytes(secret[:]).PubKey()
		require.NotEmpty(t, privateKey)
		require.NotEmpty(t, publicKey)

		address := addr.PubkeyToAddress(*publicKey.ToECDSA()).String()
		require.NotEmpty(t, address)
		t.Log(address)

		t.Log(hexutil.Encode(publicKey.SerializeCompressed()))
		t.Log(hexutil.Encode(privateKey.Serialize()))
	})
}

// TestAddressSecret - validates private key for mnemonic.
func TestAddressSecret(t *testing.T) {
	_, private, _, err := tron.WalletPubKeyHash(mnemonic, passphrase, defaultSequence)
	require.NoError(t, err)
	t.Log(private)
}

func TestGetAllChainParams(t *testing.T) {
	tc, err := tronClient(tronNodeGRPCAddr)
	require.NoError(t, err)

	params, err := tc.Client.GetChainParameters(context.Background(), &api.EmptyMessage{})
	require.NoError(t, err)

	err = testutils.PrintJSON(params)
	require.NoError(t, err)
}

func TestIsAccountActivated(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)
	defer tr.Stop(ctx)

	tests := []struct {
		address     string
		isActivated bool
	}{
		{
			// currently this address is not activated
			address:     "TRDGZSLHBdp4a2RCfY7basfKzGdxJug184",
			isActivated: false,
		},
		{
			address:     "TP3u5ojcXh3fPHeoVFXGV97kaYBJNygQ6J",
			isActivated: true,
		},
		{
			address:     "TCrRUdy9CPTFjrwv6rFShBhCDLwQ4cRJVH",
			isActivated: true,
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			isActivated, err := tr.CheckIsWalletActivated(tc.address)
			require.NoError(t, err)
			require.Equal(t, tc.isActivated, isActivated)
		})
	}
}
