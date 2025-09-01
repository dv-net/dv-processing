package evm_test

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/util"
	"github.com/dv-net/dv-processing/pkg/testutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm/erc20"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/mx/cfg"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func newClient(ctx context.Context) (*evm.EVM, error) {
	conf := new(config.Config)
	if err := cfg.Load(conf, cfg.WithLoaderConfig(cfg.Config{
		SkipFlags: true,
		Files:     []string{"../../../config.yaml"},
	})); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	c := evm.Config{
		NodeAddr:   conf.Blockchain.Ethereum.Node.Address,
		Blockchain: wconstants.BlockchainTypeEthereum,
	}

	ethService, err := evm.NewEVM(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to init eth service: %w", err)
	}

	return ethService, nil
}

func TestEstimate(t *testing.T) {
	ctx := testutils.GetContext()

	cl, err := newClient(ctx)
	require.NoError(t, err)

	gasPrice, err := cl.Node().SuggestGasPrice(ctx)
	require.NoError(t, err)

	fmt.Println("gas price", gasPrice.String())

	parsedABI, err := abi.JSON(strings.NewReader(erc20.ERC20ABI))
	require.NoError(t, err)

	tests := []struct {
		fromAddress     string
		toAddress       string
		contractAddress string
		amount          *big.Int
	}{
		{
			fromAddress:     "0xcED92FA7f0797cBc851B48140aE218a0b0D41ce0",
			toAddress:       "0x7196BA18D7aC6159C68758EeDB0639cb3e407014",
			contractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			amount:          big.NewInt(2469320000),
		},
		{
			fromAddress:     "0xcED92FA7f0797cBc851B48140aE218a0b0D41ce0",
			toAddress:       "0x7196BA18D7aC6159C68758EeDB0639cb3e407014",
			contractAddress: "eth",
			amount:          big.NewInt(2469320000),
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			var data []byte

			toAddress := common.HexToAddress(tc.toAddress)
			if tc.contractAddress != "eth" {
				toAddress = common.HexToAddress(tc.contractAddress)
				data, err = parsedABI.Pack("transfer", toAddress, tc.amount)
				require.NoError(t, err)
			}

			estimatedGas, err := cl.Node().EstimateGas(ctx, ethereum.CallMsg{
				From: common.HexToAddress(tc.fromAddress),
				To:   &toAddress,
				Data: data,
			})
			require.NoError(t, err)

			fmt.Println("estimated gas", estimatedGas)

			totalCost := new(big.Int).Mul(gasPrice, big.NewInt(int64(estimatedGas)))
			fmt.Println(decimal.NewFromBigInt(totalCost, 0).Div(decimal.NewFromInt(1e18)).String())
		})
	}
}

func TestEstimateFee(t *testing.T) {
	ctx := testutils.GetContext()

	cl, err := newClient(ctx)
	require.NoError(t, err)

	res, err := cl.EstimateFee(ctx)
	require.NoError(t, err)

	fmt.Printf("%+v\n", res)

	// fmt.Println("BaseFee:", evm.NewUnit(res.BaseFee, evm.EtherUnitWei).Value(evm.EtherUnitGWei).String())
	fmt.Println("MaxFeePerGas:", evm.NewUnit(res.MaxFeePerGas, evm.EtherUnitWei).Value(evm.EtherUnitGWei).String())
	fmt.Println("SuggestGasPrice:", evm.NewUnit(res.SuggestGasPrice, evm.EtherUnitWei).Value(evm.EtherUnitGWei).String())
	fmt.Println("SuggestGasTipCap:", evm.NewUnit(res.SuggestGasTipCap, evm.EtherUnitWei).Value(evm.EtherUnitGWei).String())
	// fmt.Println("DefaultGasTipCap:", evm.NewUnit(res.DefaultGasTipCap, evm.EtherUnitWei).Value(evm.EtherUnitGWei).String())
}

func TestEstimateContractTransfer(t *testing.T) {
	ctx := testutils.GetContext()

	cl, err := newClient(ctx)
	require.NoError(t, err)

	res, err := cl.EstimateTransfer(ctx, "0xcED92FA7f0797cBc851B48140aE218a0b0D41ce0", "0x95891645977c7b402d5a9341266ad34103895c7f", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", decimal.NewFromFloat(15.09), int64(6))
	require.NoError(t, err)

	fmt.Printf("%+v\n", res)
	fmt.Println(evm.NewUnit(res.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).String())
}

func TestEstimateETHTransfer(t *testing.T) {
	ctx := testutils.GetContext()

	cl, err := newClient(ctx)
	require.NoError(t, err)

	fromAddress := "0xbec9c6ec58a532cd8aca0af9ce28bf814651b917"
	toAddress := "0x95891645977c7b402d5a9341266ad34103895c7f"

	balanceData, err := cl.Node().BalanceAt(ctx, common.HexToAddress(fromAddress), nil)
	require.NoError(t, err)

	fmt.Println("balance", balanceData.String())

	amount, err := decimal.NewFromString(util.FormatNumberWithPrecision(balanceData.String(), 18))
	require.NoError(t, err)
	// amount := decimal.NewFromFloat(0.047925)

	res, err := cl.EstimateTransfer(ctx, fromAddress, toAddress, "eth", amount, evm.EVMAssetDecimals)
	require.NoError(t, err)

	amount = amount.Sub(evm.NewUnit(res.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).Decimal())

	fmt.Printf("%+v\n", res)
	fmt.Println(evm.NewUnit(res.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).String())

	fmt.Println(amount.String())
}

func TestPendingTX(t *testing.T) {
	ctx := testutils.GetContext()

	cl, err := newClient(ctx)
	require.NoError(t, err)

	hash := "0x8823cf841492bdd1ae972c9d71884bbbefd8f6c075d44df2fc49aec3f7da5b16"

	tx, isPending, err := cl.Node().TransactionByHash(ctx, common.HexToHash(hash))
	require.NoError(t, err)

	fmt.Println("isPending", isPending)

	testutils.PrintJSON(tx)

	txr, err := cl.Node().TransactionReceipt(ctx, common.HexToHash(hash))
	require.NoError(t, err)
	testutils.PrintJSON(txr)
}
