package processedblocks

import "fmt"

var (
	errBlockchainEmpty         = fmt.Errorf("blockchain is empty")
	errNumberLessOrEqualToZero = fmt.Errorf("number is less than or equal to 0")
)
