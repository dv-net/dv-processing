package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/stretchr/testify/require"
)

func TestValidateBTCLikeNodeAuth(t *testing.T) {
	require.NoError(t, validateBTCLikeNodeAuth("bitcoin", "", ""))
	require.NoError(t, validateBTCLikeNodeAuth("bitcoin", "rpc", "secret"))
	require.NoError(t, validateBTCLikeNodeAuth("bitcoin", "", "secret"))
	require.Error(t, validateBTCLikeNodeAuth("bitcoin", "rpc", ""))
}

func TestBTCLikeRPCAuth(t *testing.T) {
	user, pass := btcLikeRPCAuth("", "")
	require.Empty(t, user)
	require.Equal(t, btcLikeNoAuthPass, pass)

	user, pass = btcLikeRPCAuth("rpc", "secret")
	require.Equal(t, "rpc", user)
	require.Equal(t, "secret", pass)
}

// Without credentials the rpcclient must still send requests instead of
// failing on cookie auth lookup.
func TestBTCLikeNodeWithoutCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":42,"error":null,"id":1}`))
	}))
	defer srv.Close()

	var conf BitcoinBlockchain
	conf.Enabled = true
	conf.Node.Address = strings.TrimPrefix(srv.URL, "http://")
	conf.Node.UseTLS = false
	require.NoError(t, conf.Validate())

	sdkConf := conf.ConvertToSDKConfig(constants.ProcessingIdentity{})
	cl, err := btc.NewBTC(sdkConf, nil)
	require.NoError(t, err)

	count, err := cl.Node().GetBlockCount()
	require.NoError(t, err)
	require.EqualValues(t, 42, count)
}
