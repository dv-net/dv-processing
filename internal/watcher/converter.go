package watcher

import (
	"errors"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"

	commonv2 "github.com/dv-net/dv-proto/gen/go/eproxy/common/v2"
	addressesv1 "github.com/dv-net/dv-proto/gen/go/watcher/addresses/v1"
)

func (s *Service) convertHotWalletsToPb(hot []*models.HotWallet) []*addressesv1.Address {
	addresses := make([]*addressesv1.Address, 0, len(hot))
	for _, wallet := range hot {
		addr, err := s.convertHotWalletToPb(wallet)
		if err != nil {
			continue
		}
		addresses = append(addresses, addr)
	}

	return addresses
}

func (s *Service) convertHotWalletToPb(wallet *models.HotWallet) (*addressesv1.Address, error) {
	var pbBlockchain commonv2.Blockchain
	switch wallet.Blockchain {
	case wconstants.BlockchainTypeBitcoin:
		pbBlockchain = commonv2.Blockchain_BLOCKCHAIN_BITCOIN
	case wconstants.BlockchainTypeLitecoin:
		pbBlockchain = commonv2.Blockchain_BLOCKCHAIN_LITECOIN
	case wconstants.BlockchainTypeBitcoinCash:
		pbBlockchain = commonv2.Blockchain_BLOCKCHAIN_BITCOINCASH
	case wconstants.BlockchainTypeDogecoin:
		pbBlockchain = commonv2.Blockchain_BLOCKCHAIN_DOGECOIN
	default:
		return nil, errors.New("unsupported blockchain")
	}

	return &addressesv1.Address{
		Value:      wallet.Address,
		Blockchain: pbBlockchain,
	}, nil
}
