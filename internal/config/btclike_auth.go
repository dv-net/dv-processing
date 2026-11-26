package config

import "fmt"

// btcLikeNoAuthPass is sent as the RPC password when the node is configured
// without credentials. btcd/ltcd/bchd rpcclients fall back to cookie auth when
// Pass is empty and fail every request because CookiePath is not set, so a
// non-empty placeholder is required to talk to a node that does not check auth.
const btcLikeNoAuthPass = "none"

// validateBTCLikeNodeAuth allows the node to be used without credentials,
// but if a login is provided the password must be provided too.
func validateBTCLikeNodeAuth(chain, login, secret string) error {
	if login != "" && secret == "" {
		return fmt.Errorf("%s: node password must not be empty when node login is set", chain)
	}
	return nil
}

// btcLikeRPCAuth returns the user/pass pair for a btc-like rpcclient.ConnConfig.
func btcLikeRPCAuth(login, secret string) (string, string) {
	if secret == "" {
		return login, btcLikeNoAuthPass
	}
	return login, secret
}
