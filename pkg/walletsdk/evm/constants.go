package evm

const (
	// MaxGasLimit is the maximum gas limit we are willing to set for a base asset
	// transfer. A plain EOA transfer costs 21000; the headroom covers recipients
	// that execute code on receive (a contract, or an EOA with an EIP-7702
	// delegation). An on-chain estimate above this is treated as an error rather
	// than silently broadcasting an expensive or reverting transaction.
	MaxGasLimit = 150000
	MaxGasPrice = 25000000000

	// NativeTransferGasBuffer pads the on-chain gas estimate for native transfers
	// whose recipient runs code on receive. That execution is state-dependent, so
	// the raw estimate can fall short at execution time and revert with
	// "out of gas".
	NativeTransferGasBuffer = 1.5
)

const (
	EVMAssetDecimals         = 18
	TransferFeeCoeff float64 = 1.007
)
