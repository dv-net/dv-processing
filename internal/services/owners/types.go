package owners

import "github.com/dv-net/dv-processing/internal/constants"

type PrivateKeysItem struct {
	PublicKey  string
	PrivateKey string
	Address    string
	Kind       constants.WalletType
}

type PrivateKeyItem struct {
	Address    string `json:"address"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}
