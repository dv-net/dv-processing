package walletsdk

type BitcoinConfig struct {
	Network string
}

type LitecoinConfig struct {
	Network string
}

type BitcoinCashConfig struct {
	Network string
}

type DogecoinConfig struct {
	Network string
}

type Config struct {
	Bitcoin     BitcoinConfig
	Litecoin    LitecoinConfig
	BitcoinCash BitcoinCashConfig
	Dogecoin    DogecoinConfig
}
