package tron_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/dv-net/dv-processing/pkg/testutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/go-bip39"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	addr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/stretchr/testify/require"
)

const (
	mnemonic        = "vague wool express sniff alley core hen symptom end rather month cave cross elder nest bright paddle use voice wife dolphin mosquito inside curve"
	passphrase      = "" //nolint:gosec
	defaultSequence = 0
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
		secret, chainCode := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
		require.NotEmpty(t, secret)
		require.NotEmpty(t, chainCode)
		secret, err := hd.DerivePrivateKeyForPath(
			crypto.S256(),
			secret,
			chainCode,
			"44'/195'/0'/0/"+strconv.Itoa(defaultSequence),
		)
		require.NoError(t, err)
		require.NotEmpty(t, secret)

		privateKey, err := crypto.ToECDSA(secret[:])
		require.NotEmpty(t, privateKey)
		require.NoError(t, err)

		publicKey := privateKey.PublicKey

		address := addr.PubkeyToAddress(publicKey).String()
		require.NotEmpty(t, address)

		{
			walletAddress := address
			require.Equal(t, walletAddress, "TNhpnt7RTBbqHJ6KXARXNXzP3DmCLeda9t")
			walletPublicKey := hexutil.Encode(crypto.CompressPubkey(&publicKey))
			require.Equal(t, walletPublicKey, "0x022f7180d4139d93e139bb54eaea8950a6d63e73c1ccbdc5a67648ca46c24d7890")
			walletPrivateKey := hexutil.Encode(crypto.FromECDSA(privateKey))
			require.Equal(t, walletPrivateKey, "0xed809dfbae236bef30e235f4c871736205b58cf387167ded93ebcbc20865b0ef")

		}
		t.Log(address)
		t.Log(hexutil.Encode(crypto.CompressPubkey(&publicKey)))
		t.Log(hexutil.Encode(crypto.FromECDSA(privateKey)))
	})
}

// TestAddressSecret - validates private key for mnemonic.
func TestAddressSecret(t *testing.T) {
	_, private, _, err := tron.WalletPubKeyHash(mnemonic, passphrase, defaultSequence)
	require.NoError(t, err)
	t.Log(hexutil.Encode(crypto.FromECDSA(private)))
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
			address:     "TD76i6fcmbZrfWqRVgmM6aEKZCVevy4Fmv",
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
