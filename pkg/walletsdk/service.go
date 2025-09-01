package walletsdk

import (
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/dv-net/dv-processing/pkg/chainparams"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	bchchaincfg "github.com/gcash/bchd/chaincfg"
	ltcchaincfg "github.com/ltcsuite/ltcd/chaincfg"
)

const testNet = "testnet"

type SDK struct {
	BTC  *btc.WalletSDK
	LTC  *ltc.WalletSDK
	BCH  *bch.WalletSDK
	Doge *doge.WalletSDK
	Tron *tron.WalletSDK
	EVM  *evm.WalletSDK
}

func New(conf Config) *SDK {
	btcChainParams := &btcchaincfg.MainNetParams
	if conf.Bitcoin.Network == testNet {
		btcChainParams = &btcchaincfg.TestNet3Params
	}

	ltcChainParams := &ltcchaincfg.MainNetParams
	if conf.Litecoin.Network == testNet {
		ltcChainParams = &ltcchaincfg.TestNet4Params
	}

	bchChainParams := &bchchaincfg.MainNetParams
	if conf.BitcoinCash.Network == testNet {
		bchChainParams = &bchchaincfg.TestNet3Params
	}

	dogeChainParams := &chainparams.DogecoinMainNetParams
	if conf.Dogecoin.Network == testNet {
		dogeChainParams = &chainparams.DogecoinTestNet3Params
	}

	return &SDK{
		BTC:  btc.NewWalletSDK(btcChainParams),
		LTC:  ltc.NewWalletSDK(ltcChainParams),
		BCH:  bch.NewWalletSDK(bchChainParams),
		Doge: doge.NewWalletSDK(dogeChainParams),
		Tron: tron.NewWalletSDK(),
		EVM:  evm.NewWalletSDK(),
	}
}

func (s *SDK) AddressWallet(blockchain wconstants.BlockchainType, addressType string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		addrData, err := s.BTC.GenerateAddress(btc.AddressType(addressType), mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.Address.String(), nil

	case wconstants.BlockchainTypeLitecoin:
		addrData, err := s.LTC.GenerateAddress(ltc.AddressType(addressType), mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.Address.String(), nil

	case wconstants.BlockchainTypeBitcoinCash:
		addrData, err := s.BCH.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.Address.String(), nil
	case wconstants.BlockchainTypeDogecoin:
		addrData, err := s.Doge.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.Address.String(), nil

	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return evm.AddressWallet(mnemonic, passphrase, sequence)

	case wconstants.BlockchainTypeTron:
		return tron.AddressWallet(mnemonic, passphrase, sequence)

	default:
		return "", ErrBlockchainUndefined
	}
}

func (s *SDK) AddressSecret(blockchain wconstants.BlockchainType, address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		addrType, err := s.BTC.DecodeAddressType(address)
		if err != nil {
			return "", err
		}
		addrData, err := s.BTC.GenerateAddress(addrType, mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.PrivateKeyWIF.String(), nil

	case wconstants.BlockchainTypeLitecoin:
		addrType, err := s.LTC.DecodeAddressType(address)
		if err != nil {
			return "", err
		}
		addrData, err := s.LTC.GenerateAddress(addrType, mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.PrivateKeyWIF.String(), nil

	case wconstants.BlockchainTypeBitcoinCash:
		addrData, err := s.BCH.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.PrivateKeyWIF.String(), nil

	case wconstants.BlockchainTypeDogecoin:
		addrData, err := s.Doge.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}
		return addrData.PrivateKeyWIF.String(), nil
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return evm.AddressSecret(address, mnemonic, passphrase, sequence)

	case wconstants.BlockchainTypeTron:
		return tron.AddressSecret(address, mnemonic, passphrase, sequence)

	default:
		return "", ErrBlockchainUndefined
	}
}

func (s *SDK) AddressPublic(blockchain wconstants.BlockchainType, address string, mnemonic string, passphrase string, sequence uint32) (string, error) {
	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		addrType, err := s.BTC.DecodeAddressType(address)
		if err != nil {
			return "", err
		}

		addrData, err := s.BTC.GenerateAddress(addrType, mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return "", err
		}

		return pubKey, nil

	case wconstants.BlockchainTypeLitecoin:
		addrType, err := s.LTC.DecodeAddressType(address)
		if err != nil {
			return "", err
		}

		addrData, err := s.LTC.GenerateAddress(addrType, mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return "", err
		}

		return pubKey, nil

	case wconstants.BlockchainTypeBitcoinCash:
		addrData, err := s.BCH.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return "", err
		}

		return pubKey, nil

	case wconstants.BlockchainTypeDogecoin:
		addrData, err := s.Doge.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return "", err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return "", err
		}

		return pubKey, nil
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return evm.AddressPublic(address, mnemonic, passphrase, sequence)

	case wconstants.BlockchainTypeTron:
		return tron.AddressPublic(address, mnemonic, passphrase, sequence)

	default:
		return "", ErrBlockchainUndefined
	}
}

type GenerateAddress struct {
	Address    string
	PublicKey  string
	PrivateKey string
}

func (s *SDK) GenerateAddress(blockchain wconstants.BlockchainType, addressType string, mnemonic string, passphrase string, sequence uint32) (*GenerateAddress, error) {
	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		addrData, err := s.BTC.GenerateAddress(btc.AddressType(addressType), mnemonic, passphrase, sequence)
		if err != nil {
			return nil, err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return nil, err
		}

		return &GenerateAddress{
			Address:    addrData.Address.String(),
			PublicKey:  pubKey,
			PrivateKey: addrData.PrivateKeyWIF.String(),
		}, nil

	case wconstants.BlockchainTypeLitecoin:
		addrData, err := s.LTC.GenerateAddress(ltc.AddressType(addressType), mnemonic, passphrase, sequence)
		if err != nil {
			return nil, err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return nil, err
		}

		return &GenerateAddress{
			Address:    addrData.Address.String(),
			PublicKey:  pubKey,
			PrivateKey: addrData.PrivateKeyWIF.String(),
		}, nil

	case wconstants.BlockchainTypeBitcoinCash:
		addrData, err := s.BCH.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return nil, err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return nil, err
		}

		return &GenerateAddress{
			Address:    addrData.Address.String(),
			PublicKey:  pubKey,
			PrivateKey: addrData.PrivateKeyWIF.String(),
		}, nil
	case wconstants.BlockchainTypeDogecoin:
		addrData, err := s.Doge.GenerateAddress(mnemonic, passphrase, sequence)
		if err != nil {
			return nil, err
		}

		pubKey, err := addrData.AddressPubKey()
		if err != nil {
			return nil, err
		}

		return &GenerateAddress{
			Address:    addrData.Address.String(),
			PublicKey:  pubKey,
			PrivateKey: addrData.PrivateKeyWIF.String(),
		}, nil

	default:
		return nil, ErrBlockchainUndefined
	}
}

func (s *SDK) ValidateAddress(blockchain wconstants.BlockchainType, address string) bool {
	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		return s.BTC.ValidateAddress(address)

	case wconstants.BlockchainTypeLitecoin:
		return s.LTC.ValidateAddress(address)

	case wconstants.BlockchainTypeBitcoinCash:
		return s.BCH.ValidateAddress(address)

	case wconstants.BlockchainTypeDogecoin:
		return s.Doge.ValidateAddress(address)

	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return evm.ValidateAddress(address)

	case wconstants.BlockchainTypeTron:
		return tron.ValidateAddress(address)

	default:
		return false
	}
}
