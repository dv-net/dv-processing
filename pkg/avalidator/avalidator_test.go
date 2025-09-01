package avalidator_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dv-net/dv-processing/pkg/avalidator"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	// Examples of addresses for verification
	addresses := []struct {
		addr       string
		network    string
		blockchain string
	}{
		{"144doNUHK6VjbjJBUeAgMX2WvKqBxLtNow", "mainnet", "bitcoin"},                             // Bitcoin main network (P2PKH)
		{"3BvMNMiV74tWciG2LXLUmrFunQaEbM4yCK", "mainnet", "bitcoin"},                             // Bitcoin main network (P2SH)
		{"bc1q5y7mkzgz7r4045ee2x3fjat5pyjlwejjlzszxf", "mainnet", "bitcoin"},                     // Bitcoin main network (bech32)
		{"mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn", "testnet", "bitcoin"},                             // Bitcoin test network (P2PKH)
		{"2NBFNJTktNa7GZusGbDbGKRZTxdK9VVez3n", "testnet", "bitcoin"},                            // Bitcoin test network (P2SH)
		{"tb1p0924wz2pcfap83dw7q345uuu0yshgw6jvvmvetpfkstulwdy92nsk3r4af", "testnet", "bitcoin"}, // Bitcoin test network (bech32)
		{"LM8xnzY9UNEp4NSumEU8B4PeSLGjoQ6iFb", "mainnet", "litecoin"},                            // Litecoin main network
		{"MUxgJPhiEPcGxymiUEJjKZKKJTE3hWEYkz", "mainnet", "litecoin"},                            // Litecoin main network
		{"ltc1qx2afhw92dmkxhj7ya2guuzh43a03l33gsuecce", "mainnet", "litecoin"},                   // Litecoin main network
		{"0x32Be343B94f860124dC4fEe278FDCBD38C102D88", "mainnet", "ethereum"},                    // Ethereum main network
		{"0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359", "testnet", "ethereum"},                    // Ethereum test network
		{"TG8Td7yY8mQFryAPepshgVrJy1ZxjXLqqk", "mainnet", "tron"},                                // Tron main network
		{"TYAKY4oBuNbhd7po5XTT6pmCNGv1Fkz9me", "testnet", "tron"},                                // Tron test network
		{"DEgDVFa2DoW1533dxeDVdTxQFhMzs1pMke", "mainnet", "dogecoin"},                            // Dogeoin main network (P2PKH)
		{"9srEbLELgnH8rQ69Mcb35es2p68aeG9fZw", "mainnet", "dogecoin"},                            // Dogeoin main network (P2SH)
		{"njyMWWyh1L7tSX6QkWRgetMVCVyVtfoDta", "testnet", "dogecoin"},                            // Dogeoin testnet (P2PKH)
	}

	// Checking addresses
	for _, a := range addresses {
		t.Run("TestValidator", func(t *testing.T) {
			var res bool
			switch strings.ToLower(a.blockchain) {
			case "bitcoin":
				res = avalidator.ValidateBitcoinAddress(a.addr)
				fmt.Printf("Bitcoin address %s (%s): %v\n", a.addr, a.network, res)
			case "litecoin":
				res = avalidator.ValidateLitecoinAddress(a.addr)
				fmt.Printf("Litecoin address %s (%s): %v\n", a.addr, a.network, res)
			case "ethereum":
				res = avalidator.ValidateEVMAddress(a.addr)
				fmt.Printf("Ethereum address %s (%s): %v\n", a.addr, a.network, res)
			case "tron":
				res = avalidator.ValidateTronAddress(a.addr)
				fmt.Printf("Tron address %s (%s): %v\n", a.addr, a.network, res)
			case "dogecoin":
				res = avalidator.ValidateDogecoinAddress(a.addr)
				fmt.Printf("Dogecoin address %s (%s): %v\n", a.addr, a.network, res)
			default:
				t.Fatalf("Unsupported blockchain: %s", a.blockchain)
			}

			require.Equal(t, true, res)
		})
	}
}
