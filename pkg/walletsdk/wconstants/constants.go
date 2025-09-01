package wconstants

type BlockchainType string

type Blockchains []BlockchainType

// Strings returns the blockchain types as a slice of strings
func (b Blockchains) Strings() []string {
	strings := make([]string, len(b))
	for i, blockchain := range b {
		strings[i] = blockchain.String()
	}
	return strings
}

const (
	// Tron blockchain
	BlockchainTypeTron BlockchainType = "tron"

	// EVM blockchains
	BlockchainTypeEthereum          BlockchainType = "ethereum"
	BlockchainTypeBinanceSmartChain BlockchainType = "bsc"
	BlockchainTypePolygon           BlockchainType = "polygon"
	BlockchainTypeArbitrum          BlockchainType = "arbitrum"
	BlockchainTypeOptimism          BlockchainType = "optimism"
	BlockchainTypeLinea             BlockchainType = "linea"

	// Bitcoin-like blockchains
	BlockchainTypeBitcoin     BlockchainType = "bitcoin"
	BlockchainTypeLitecoin    BlockchainType = "litecoin"
	BlockchainTypeBitcoinCash BlockchainType = "bitcoincash"
	BlockchainTypeDogecoin    BlockchainType = "dogecoin"
)

// AllBlockchainsStrings returns all blockchain types as a slice of strings
var AllBlockchains = Blockchains{
	BlockchainTypeTron,
	BlockchainTypeEthereum,
	BlockchainTypeBinanceSmartChain,
	BlockchainTypePolygon,
	BlockchainTypeArbitrum,
	BlockchainTypeOptimism,
	BlockchainTypeLinea,
	BlockchainTypeBitcoin,
	BlockchainTypeLitecoin,
	BlockchainTypeBitcoinCash,
	BlockchainTypeDogecoin,
}

// Validate checks if the blockchain type is valid
func (b BlockchainType) Valid() bool {
	switch b {
	case
		BlockchainTypeTron,
		BlockchainTypeEthereum,
		BlockchainTypeBinanceSmartChain,
		BlockchainTypePolygon,
		BlockchainTypeArbitrum,
		BlockchainTypeOptimism,
		BlockchainTypeLinea,
		BlockchainTypeBitcoin,
		BlockchainTypeLitecoin,
		BlockchainTypeBitcoinCash,
		BlockchainTypeDogecoin:
		return true
	}
	return false
}

// String returns the blockchain type as a string
func (b BlockchainType) String() string { return string(b) }

func (b BlockchainType) GetAssetIdentifier() string {
	switch b {
	case BlockchainTypeTron:
		return "trx"

	case BlockchainTypeEthereum:
		return "eth"
	case BlockchainTypeBinanceSmartChain:
		return "bnb"
	case BlockchainTypePolygon:
		return "pol"
	case BlockchainTypeArbitrum,
		BlockchainTypeOptimism,
		BlockchainTypeLinea:
		return "eth"

	case BlockchainTypeBitcoin:
		return "btc"
	case BlockchainTypeLitecoin:
		return "ltc"
	case BlockchainTypeBitcoinCash:
		return "bch"
	case BlockchainTypeDogecoin:
		return "doge"
	default:
		return ""
	}
}

func (b BlockchainType) IsEVM() bool {
	switch b {
	case BlockchainTypeEthereum,
		BlockchainTypeBinanceSmartChain,
		BlockchainTypePolygon,
		BlockchainTypeArbitrum,
		BlockchainTypeOptimism,
		BlockchainTypeLinea:
		return true
	default:
		return false
	}
}

func EVMBlockchains() Blockchains {
	return Blockchains{
		BlockchainTypeEthereum,
		BlockchainTypeBinanceSmartChain,
		BlockchainTypePolygon,
		BlockchainTypeArbitrum,
		BlockchainTypeOptimism,
		BlockchainTypeLinea,
	}
}

func (b BlockchainType) IsBitcoinLike() bool {
	switch b {
	case BlockchainTypeBitcoin,
		BlockchainTypeLitecoin,
		BlockchainTypeBitcoinCash,
		BlockchainTypeDogecoin:
		return true
	default:
		return false
	}
}

func (b BlockchainType) IsSystemTransactionsSupported() bool {
	return b.IsEVM() || b == BlockchainTypeTron
}
