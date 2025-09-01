package doge

import (
	"encoding/hex"
	"math/big"
	"time"

	btcchaincfg "github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/chaincfg/chainhash"
)

// DogecoinMainNetParams defines the network parameters for Dogecoin Mainnet.
var DogecoinMainNetParams = btcchaincfg.Params{
	Name:        "dogecoin",
	Net:         0xc0c0c0c0,
	DefaultPort: "22556",
	DNSSeeds: []btcchaincfg.DNSSeed{
		{Host: "seed.dogecoin.com", HasFiltering: true},
		{Host: "seed2.dogecoin.com", HasFiltering: true},
		{Host: "seed.multidoge.org", HasFiltering: true},
		{Host: "seed2.multidoge.org", HasFiltering: true},
		{Host: "seed.moolah.io", HasFiltering: true},
		{Host: "seed.vdigger.com", HasFiltering: true},
	},
	GenesisHash:              newHashFromStr("1a91e3dace36e2be3bf030a65679fe821aa1d6ef92e7c9902eb318182c355691"),
	PowLimit:                 new(big.Int).SetBytes(hexDecode("00000fffffffffffffffffffffffffffffffffffffffffffffffffffffff")),
	TargetTimespan:           time.Hour * 4,
	TargetTimePerBlock:       time.Minute,
	RetargetAdjustmentFactor: 4,
	CoinbaseMaturity:         30,
	SubsidyReductionInterval: 100000,
	PubKeyHashAddrID:         0x1e,
	ScriptHashAddrID:         0x16,
	PrivateKeyID:             0x9e,
	HDPrivateKeyID:           [4]byte{0x02, 0xfa, 0xc3, 0x98},
	HDPublicKeyID:            [4]byte{0x02, 0xfa, 0xca, 0xfd},
	HDCoinType:               3,
	Bech32HRPSegwit:          "",
	BIP0034Height:            21111,
	BIP0065Height:            22857,
	BIP0066Height:            91812,
}

// DogecoinTestNet3Params defines the network parameters for Dogecoin Testnet.
var DogecoinTestNet3Params = btcchaincfg.Params{
	Name:        "dogecoin-testnet",
	Net:         0xfcc1b7dc,
	DefaultPort: "44556",
	DNSSeeds: []btcchaincfg.DNSSeed{
		{Host: "testnet-seed.dogecoin.com"},
		{Host: "testnet-seed.multidoge.org"},
	},
	GenesisHash:              newHashFromStr("bb0a78264637406b6360aad926e6b0c2f20a3e6f914716cd76606cbeb7df48ab"),
	PowLimit:                 new(big.Int).SetBytes(hexDecode("00000fffffffffffffffffffffffffffffffffffffffffffffffffffffff")),
	TargetTimespan:           time.Hour * 4,
	TargetTimePerBlock:       time.Minute,
	RetargetAdjustmentFactor: 4,
	CoinbaseMaturity:         30,
	SubsidyReductionInterval: 100000,
	PubKeyHashAddrID:         0x71,
	ScriptHashAddrID:         0xc4,
	PrivateKeyID:             0xf1,
	HDPrivateKeyID:           [4]byte{0x04, 0x35, 0x83, 0x94},
	HDPublicKeyID:            [4]byte{0x04, 0x35, 0x87, 0xcf},
	HDCoinType:               3,
	Bech32HRPSegwit:          "",
	BIP0034Height:            1,
	BIP0065Height:            1,
	BIP0066Height:            1,
}

func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, _ := chainhash.NewHashFromStr(hexStr)
	return hash
}

func hexDecode(hexStr string) []byte {
	b, _ := hex.DecodeString(hexStr)
	return b
}
