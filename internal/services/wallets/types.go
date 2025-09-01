package wallets

import (
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

type PrivateKeysItem struct {
	PublicKey  string
	PrivateKey string
	Address    string
	Kind       constants.WalletType
}

type GetAllPrivateKeysResponse map[wconstants.BlockchainType][]PrivateKeysItem
