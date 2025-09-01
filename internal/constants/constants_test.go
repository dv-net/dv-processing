package constants_test

import (
	"fmt"
	"testing"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

func TestConfirmationsTimeoutWithRequired(t *testing.T) {
	res := constants.ConfirmationsTimeoutWithRequired(wconstants.BlockchainTypeEthereum, 2, 1)
	fmt.Println(res)
}
