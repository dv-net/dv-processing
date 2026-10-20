package tron_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
)

func TestCheckProcessingBandwidth(t *testing.T) {
	d := decimal.NewFromInt

	tests := []struct {
		name      string
		need      tron.EstimateTransferWithDelegateResourcesResponse
		available tron.Resources
		reserved  decimal.Decimal
		wantErr   bool
	}{
		{
			name: "energy only delegation is paid by free bandwidth",
			need: tron.EstimateTransferWithDelegateResourcesResponse{
				NeedBandwidthFromProcessingWallet: d(560),
			},
			available: tron.Resources{Bandwidth: d(0), FreeBandwidth: d(600)},
		},
		{
			name: "not enough staked plus free",
			need: tron.EstimateTransferWithDelegateResourcesResponse{
				NeedBandwidthFromProcessingWallet: d(560),
			},
			available: tron.Resources{Bandwidth: d(100), FreeBandwidth: d(400)},
			wantErr:   true,
		},
		{
			name: "delegated bandwidth needs staked bandwidth",
			need: tron.EstimateTransferWithDelegateResourcesResponse{
				NeedBandwidthFromProcessingWallet:       d(345 + 560),
				NeedStakedBandwidthFromProcessingWallet: d(345 + 280),
			},
			available: tron.Resources{Bandwidth: d(500), FreeBandwidth: d(600)},
			wantErr:   true,
		},
		{
			name: "reclaim is paid by free bandwidth",
			need: tron.EstimateTransferWithDelegateResourcesResponse{
				NeedBandwidthFromProcessingWallet:       d(345 + 560),
				NeedStakedBandwidthFromProcessingWallet: d(345 + 280),
			},
			available: tron.Resources{Bandwidth: d(700), FreeBandwidth: d(600)},
		},
		{
			name: "active transfers reserve staked bandwidth",
			need: tron.EstimateTransferWithDelegateResourcesResponse{
				NeedBandwidthFromProcessingWallet:       d(345 + 560),
				NeedStakedBandwidthFromProcessingWallet: d(345 + 280),
			},
			available: tron.Resources{Bandwidth: d(700), FreeBandwidth: d(600)},
			reserved:  d(100),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.need.CheckProcessingBandwidth(tc.available, tc.reserved)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
