package util_test

import (
	"testing"

	"github.com/dv-net/dv-processing/internal/util"
	"github.com/stretchr/testify/require"
)

func TestGetByPath(t *testing.T) {
	type estimatedData struct {
		TotalFeeAmount int `json:"total_fee_amount"`
	}

	res, err := util.GetByPath[estimatedData](map[string]any{
		"send_result": map[string]any{
			"estimated_data": estimatedData{
				TotalFeeAmount: 100,
			},
		},
	}, "send_result.estimated_data")
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, estimatedData{
		TotalFeeAmount: 100,
	}, res)

	res2, err := util.GetByPath[int](map[string]any{
		"send_result": map[string]any{
			"estimated_data": estimatedData{
				TotalFeeAmount: 100,
			},
		},
	}, "send_result.estimated_data.total_fee_amount")
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 100, res2)
}
