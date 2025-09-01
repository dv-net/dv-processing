package bch_test

import (
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	"github.com/stretchr/testify/require"
)

func TestGenerateSegwit(t *testing.T) {
	addresses, err := generateTestSegwitAddresses(5)
	if err != nil {
		t.Fatalf("generate test segwit addresses: %v", err)
	}

	for _, address := range addresses {
		pubKey, err := address.AddressPubKey()
		require.NoError(t, err)

		t.Logf("Address: %s", address.Address.String())
		t.Logf("Public: %s", pubKey)
		t.Logf("Private: %s", address.PrivateKeyWIF.String())
	}
}

func TestGenerateAddress(t *testing.T) {
	addr := "qpklt0aa6vj87tqj60q0u5gy34ecqjy7auqkvwl70s"
	bchAddr, err := bchutil.DecodeAddress(addr, &chaincfg.MainNetParams)
	require.NoError(t, err)

	require.Equal(t, bchAddr.String(), addr)

	sdk := bch.NewWalletSDK(&chaincfg.MainNetParams)

	addrData, err := sdk.GenerateAddress(mnemonic, passphrase, 0)
	require.NoError(t, err)

	require.Equal(t, addrData.Address.String(), addr)

	publ, priv, err := sdk.AddressFromPrivateKey(addrData.PrivateKeyWIF.String())
	require.NoError(t, err)

	require.Equal(t, addr, publ)
	require.Equal(t, priv, addrData.PrivateKeyWIF.PrivKey)
}

func TestDecodeAddress(t *testing.T) {
	// https://github.com/gcash/bchutil/blob/master/address_test.go
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			want:  "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
		},
		{
			input: "bitcoincash:qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			want:  "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
		},
		{
			input: "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
			want:  "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
		},
		{
			input: "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
			want:  "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
		},
		{
			input: "02192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4",
			want:  "02192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4",
		},
		{
			input: "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
			want:  "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
		},
		{
			input: "bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
			want:  "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bchAddr, err := bchutil.DecodeAddress(tt.input, &chaincfg.MainNetParams)
			require.NoError(t, err)
			require.Equal(t, tt.want, bchAddr.String())
		})
	}
}

func TestConvertLegacyToCashAddr(t *testing.T) {
	params := chaincfg.MainNetParams

	tests := []struct {
		input string
		want  string
	}{
		{
			// Legacy address (P2PKH), its CASHADDR representation without a prefix is expected
			input: "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
			want:  "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
		},
		{
			// cashaddr without a prefix - the function simply normalizes it
			input: "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			want:  "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
		},
		{
			// cashaddr with prefix - prefix is deleted
			input: "bitcoincash:qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			want:  "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
		},
		{
			// Legacy address (P2SH), its CASHADDR representation without a prefix is expected
			input: "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
			want:  "pruptvpkmxamee0f72sq40gm70wfr624zq0yyxtycm",
		},
		{
			// cashaddr without a prefix - the function simply normalizes it
			input: "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
			want:  "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
		},
		{
			//  cashaddr with prefix - prefix is deleted
			input: "bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
			want:  "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := bch.DecodeAddressToCashAddr(tt.input, &params)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeAddressToLegacyAddr(t *testing.T) {
	params := chaincfg.MainNetParams

	tests := []struct {
		input string
		want  string
	}{
		{
			// cashAddr (P2PKH), its legacy is expected
			input: "bitcoincash:qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			want:  "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
		},
		{
			// cashAddr without a prefix (P2PKH), its Legacy is expected
			input: "qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			want:  "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
		},
		{
			// Legacy address (P2PKH), the function simply normalizes it
			input: "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
			want:  "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
		},
		{
			// cashAddr (P2SH), its legacy is expected
			input: "bitcoincash:pruptvpkmxamee0f72sq40gm70wfr624zq0yyxtycm",
			want:  "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
		},
		{
			// cashAddr without a prefix (P2SH), its legacy is expected
			input: "pruptvpkmxamee0f72sq40gm70wfr624zq0yyxtycm",
			want:  "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
		},
		{
			// Legacy address (P2SH), the function simply normalizes it
			input: "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
			want:  "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := bch.DecodeAddressToLegacyAddr(tt.input, &params)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsLegacyAddress(t *testing.T) {
	params := chaincfg.MainNetParams

	tests := []struct {
		input    string
		expected bool
	}{
		{
			// legacy-address (P2PKH)
			input:    "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
			expected: true,
		},
		{
			// legacy-address (P2SH)
			input:    "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
			expected: true,
		},
		{
			// cashAddr (P2PKH)
			input:    "bitcoincash:qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			expected: false,
		},
		{
			// cashAddr (P2SH)
			input:    "bitcoincash:pruptvpkmxamee0f72sq40gm70wfr624zq0yyxtycm",
			expected: false,
		},
		{
			// invalid address
			input:    "invalidaddress",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := bch.IsLegacyAddress(tt.input, &params)
			if tt.input == "invalidaddress" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCashAddrAddress(t *testing.T) {
	params := chaincfg.MainNetParams

	tests := []struct {
		input    string
		expected bool
	}{
		{
			// cashAddr (P2PKH)
			input:    "bitcoincash:qp4vfhxjw8kxannnmemdvxpnkndx2mguf5zfge37ru",
			expected: true,
		},
		{
			// cashAddr (P2SH)
			input:    "bitcoincash:pruptvpkmxamee0f72sq40gm70wfr624zq0yyxtycm",
			expected: true,
		},
		{
			// legacy-address (P2PKH)
			input:    "1AjYTb1LAAde8KTGWwvME2wc6MgpYxkPsk",
			expected: false,
		},
		{
			// legacy-address (P2SH)
			input:    "3QJmV3qfvL9SuYo34YihAf3sRCW3qSinyC",
			expected: false,
		},
		{
			// invalid address
			input:    "invalidaddress",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := bch.IsCashAddrAddress(tt.input, &params)
			if tt.input == "invalidaddress" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expected, result)
		})
	}
}
