package tron_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/testutils"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
)

var tronNodeGRPCAddr = "3.225.171.164:50051"

func tronClient(addr string) (*client.GrpcClient, error) {
	c := client.NewGrpcClientWithTimeout(addr, time.Second*30)
	err := c.Start(grpc.WithTransportCredentials(insecure.NewCredentials()))
	return c, err
}

func newTronSDK(addr string) (*tron.Tron, error) {
	if addr == "" {
		return nil, fmt.Errorf("node gRPC address must not be empty")
	}

	ctx := testutils.GetContext()

	identity, err := constants.IdentityFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get processing identity: %w", err)
	}

	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(tron.PrepareUnaryInterceptor(
			constants.ProcessingIDParamName.String(),
			identity.ID,
			constants.ProcessingVersionParamName.String(),
			identity.Version,
		)),
		grpc.WithStreamInterceptor(tron.PrepareStreamInterceptor(
			constants.ProcessingIDParamName.String(),
			identity.ID,
			constants.ProcessingVersionParamName.String(),
			identity.Version,
		),
		),
	}

	return tron.NewTron(tron.Config{
		NodeAddr:                  tronNodeGRPCAddr,
		UseTLS:                    false,
		GRPCOptions:               opts,
		ActivationContractAddress: "TVwfUJiY9g6XP8gYdETJMdmp1Cbgo9FNm7",
		UseBurnTRXActivation:      true,
	})
}

func TestAccountResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	resources, err := tr.TotalAvailableResources("TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF")
	require.NoError(t, err)
	t.Log("Total energy:" + resources.TotalEnergy.String())
	t.Log("Available energy:" + resources.Energy.String())
	t.Log("Total bandwidth:" + resources.TotalBandwidth.String())
	t.Log("Available bandwidth:" + resources.Bandwidth.String())
}

func TestGetAccount(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	data, err := tr.Node().GetAccount("TP3u5ojcXh3fPHeoVFXGV97kaYBJNygQ6J")
	require.NoError(t, err)

	testutils.PrintJSON(data)
}

func TestResourcesCalculator(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	accountresources := new(api.AccountResourceMessage)
	accountresources.EnergyLimit = 8917001
	accountresources.EnergyUsed = 8764747
	accountresources.NetLimit = 223699
	accountresources.TotalNetLimit = 432e8
	accountresources.NetUsed = 85189
	accountresources.FreeNetLimit = 600
	accountresources.FreeNetUsed = 100

	// check AvailableEnergy
	availableEnergy := tr.AvailableEnergy(accountresources)
	require.Equal(t, "152254", availableEnergy.String())

	// check AvailableBandwidth
	availableBandwidth := tr.AvailableBandwidth(accountresources)
	require.Equal(t, "139010", availableBandwidth.String())

	// check ConvertStackedTRXToEnergy
	energy := tr.ConvertStackedTRXToEnergy(180_000_000_000, 7640542262, 6685451869)
	require.Equal(t, "78749.733672", energy.String())

	// check ConvertStackedTRXToBandwidth
	bandwidth := tr.ConvertStackedTRXToBandwidth(38574173764, accountresources.TotalNetLimit, 169746280791)
	require.Equal(t, "190102.30458768", bandwidth.String())
}

func TestAvailableForDelegateResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	tests := []struct {
		address string
	}{
		{
			address: "TNUsTEX4YqorT1GsAJuZX4h7VYREjy7D66",
		},
		{
			address: "TQ6DkBmxz3Zk7neh8mwmmkfJsVjrE9wwjY",
		},
		{
			address: "TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF",
		},
		{
			address: "TWhZihYZXv4vbxAjXCQfuZQ4PF8WLzqeJe",
		},
	}

	for _, tc := range tests {
		fmt.Println("check for", tc.address)

		// check AvailableForDelegateResources
		resources, err := tr.AvailableForDelegateResources(ctx, tc.address)
		require.NoError(t, err)

		maxOperationsCount := tr.GetMaxOperationsCount(resources.Energy.IntPart(), resources.Bandwidth.IntPart())

		fmt.Println("total energy", resources.TotalEnergy.String())
		fmt.Println("available for delegate energy", resources.Energy.String())
		fmt.Println("total bandwidth", resources.TotalBandwidth.String())
		fmt.Println("available for delegate bandwidth", resources.Bandwidth.String())
		fmt.Println("maxOperationsCount", maxOperationsCount)
		fmt.Println()
	}
}

func TestCalculateQueueCapacity(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	tests := []struct {
		availableEnergy    int64
		availableBandwidth int64
		expectedCapacity   uint32
	}{
		{
			availableEnergy:    0,
			availableBandwidth: 0,
			expectedCapacity:   0,
		},
		{
			availableEnergy:    100000,
			availableBandwidth: 0,
			expectedCapacity:   0,
		},
		{
			availableEnergy:    0,
			availableBandwidth: 1000,
			expectedCapacity:   0,
		},
		{
			availableEnergy:    100,
			availableBandwidth: 100,
			expectedCapacity:   0,
		},
		{
			availableEnergy:    80000,
			availableBandwidth: 1000,
			expectedCapacity:   1,
		},
		{
			availableEnergy:    80000,
			availableBandwidth: 2000,
			expectedCapacity:   1,
		},
		{
			availableEnergy:    160000,
			availableBandwidth: 1000,
			expectedCapacity:   1,
		},
		{
			availableEnergy:    160000,
			availableBandwidth: 2000,
			expectedCapacity:   2,
		},
		{
			availableEnergy:    9078640,
			availableBandwidth: 139052,
			expectedCapacity:   113,
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := tr.GetMaxOperationsCount(tt.availableEnergy, tt.availableBandwidth)
			if got != tt.expectedCapacity {
				t.Fatalf("CalculateQueueCapacity() = %v, want %v", got, tt.expectedCapacity)
			}
		})
	}
}

func TestEstimateEnergyForTransfer(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	chainParams, err := tr.ChainParams(context.Background())
	require.NoError(t, err)

	amount := decimal.NewFromFloat(582411180)
	jsonString := fmt.Sprintf(`[{"address":"%s"},{"uint256":"%s"}]`, "TJQkYnJNn5fMPjQirvVzCufbepc9MadXJt", amount.BigInt())

	data, err := tr.Node().TriggerConstantContract("TALzgWqRfjrLFPfs3LsGfNasVUHEagmZGu", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", "transfer(address,uint256)", jsonString)
	require.NoError(t, err)

	tx := data.GetTransaction()
	tx.RawData.FeeLimit = 30_000_000
	tx.Ret = nil

	rawData, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)

	h256h := sha256.New()
	_, err = h256h.Write(rawData)
	require.NoError(t, err)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	signature, err := crypto.Sign(h256h.Sum(nil), pk)
	require.NoError(t, err)

	tx.Signature = append(tx.Signature, signature)

	energyFee := tron.NewEnergy(decimal.NewFromInt(data.EnergyUsed)).ToTRX(chainParams.EnergyFee)
	bandwidthFee := tron.NewBandwidth(decimal.NewFromInt(int64(proto.Size(tx))).Add(decimal.NewFromInt(64)))

	fmt.Println(data.EnergyUsed)
	fmt.Println(energyFee)
	fmt.Println(bandwidthFee.ToDecimal())
	fmt.Println(energyFee.ToDecimal().Add(bandwidthFee.ToTRX(1e3).ToDecimal()))
}

func TestEstimateTransferResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	tests := []struct {
		fromAddress     string
		toAddress       string
		contractAddress string
		amount          decimal.Decimal
		expected        tron.EstimateTransferResourcesResult
	}{
		{
			fromAddress:     "TPY6BootwZAmSW2zLNCyrJ21N3tQBfpxi8",
			toAddress:       "TNqNrJWP8bvjeDA2rDAjFFWxQLr8wQ28xt",
			contractAddress: "trx",
			amount:          decimal.NewFromInt(1000000),
			expected: tron.EstimateTransferResourcesResult{
				Energy:    decimal.NewFromInt(0),
				Bandwidth: decimal.NewFromInt(267),
				Trx:       decimal.NewFromFloat(0.267),
			},
		},
		{
			fromAddress:     "TU1eeQd99kuyxzYQsvzzPoiUFxXtCPnSGM",
			toAddress:       "TUV9KAug41pVonNc7CWUPtWKRBrtQC6rju",
			contractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			amount:          decimal.NewFromInt(582411180),
			expected: tron.EstimateTransferResourcesResult{
				Energy:    decimal.NewFromInt(64895),
				Bandwidth: decimal.NewFromInt(345),
				Trx:       decimal.NewFromFloat(27.6009),
			},
		},
		{
			fromAddress:     "TALzgWqRfjrLFPfs3LsGfNasVUHEagmZGu",
			toAddress:       "TJQkYnJNn5fMPjQirvVzCufbepc9MadXJt",
			contractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			amount:          decimal.NewFromInt(1853960000),
			expected: tron.EstimateTransferResourcesResult{
				Energy:    decimal.NewFromInt(31895),
				Bandwidth: decimal.NewFromInt(345),
				Trx:       decimal.NewFromFloat(13.7409),
			},
		},
		{
			fromAddress:     "TWAVZJw6jdovKqT6pdmdoLyTu2gnLhJUgh",
			toAddress:       "THsxNHeVCR7fEVwnsNoW8XAdqHgjzADXxA",
			contractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			amount:          decimal.NewFromInt(700000000),
			expected: tron.EstimateTransferResourcesResult{
				Energy:    decimal.NewFromInt(64895),
				Bandwidth: decimal.NewFromInt(345),
				Trx:       decimal.NewFromFloat(27.6009),
			},
		},
		{
			fromAddress:     "TSFX1uDdkqhkYxFdDa839FXXAFLpXk1dTT",
			toAddress:       "TJYFcrNrFoZULmtc68v7r9QR8FsVzkcDts",
			contractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			amount:          decimal.NewFromInt(20000000),
			expected: tron.EstimateTransferResourcesResult{
				Energy:    decimal.NewFromInt(31895),
				Bandwidth: decimal.NewFromInt(345),
				Trx:       decimal.NewFromFloat(13.7409),
			},
		},
		{
			fromAddress:     "TNGFDtFrH6HphWFbVW1D52UMb5rZj2Tk7F",
			toAddress:       "TBRKTUtaRp3B53W3wEhgcdo6UuqKLAJEM8",
			contractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			amount:          decimal.NewFromInt(20000000),
			expected: tron.EstimateTransferResourcesResult{
				Energy:    decimal.NewFromInt(8624),
				Bandwidth: decimal.NewFromInt(345),
				Trx:       decimal.NewFromFloat(2.15604),
			},
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			res, err := tr.EstimateTransferResources(context.Background(), tc.fromAddress, tc.toAddress, tc.contractAddress, tc.amount, 6)
			require.NoError(t, err)

			require.NotNil(t, res)

			fmt.Println("Energy:", res.Energy)
			fmt.Println("Bandwidth:", res.Bandwidth)
			fmt.Println("TRX:", res.Trx)

			require.Equal(t, tc.expected.Energy.String(), res.Energy.String())
			require.Equal(t, tc.expected.Bandwidth.String(), res.Bandwidth.String())
			require.Equal(t, tc.expected.Trx.String(), res.Trx.String())
		})
	}
}

func TestEstimateActivateFee(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	tests := []struct {
		fromAddress     string
		toAddress       string
		assetIdentifier string
		expectedFee     decimal.Decimal
	}{
		{
			fromAddress:     "TG3bVVPCouQzQNwp2uhNqcFWi19UrntBQt",
			toAddress:       "TCQKzPj37BAzS3ARQZMPFqoueFxjMii2Kv",
			assetIdentifier: tron.TrxAssetIdentifier,
			expectedFee:     decimal.NewFromFloat(0),
		},
		{
			fromAddress:     "TG3bVVPCouQzQNwp2uhNqcFWi19UrntBQt",
			toAddress:       "TFKcS7hpTf4mxXhAd9Kt883PmGwqYRuZrA",
			assetIdentifier: tron.TrxAssetIdentifier,
			expectedFee:     decimal.NewFromFloat(1),
		},
		{
			fromAddress:     "TS4cZHDNxYgD5X3m1f3yAvqLM54S8Ab2tT",
			toAddress:       "TFKcS7hpTf4mxXhAd9Kt883PmGwqYRuZrA",
			assetIdentifier: tron.TrxAssetIdentifier,
			expectedFee:     decimal.NewFromFloat(1.1),
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			var baseAmount decimal.Decimal
			account, err := tr.Node().GetAccount(tc.fromAddress)
			require.NoError(t, err)

			if tc.assetIdentifier == tron.TrxAssetIdentifier {
				baseAmount = decimal.NewFromInt(account.Balance).Div(decimal.NewFromInt(1_000_000))
			}

			if !baseAmount.IsPositive() {
				t.Fatalf("account balance is not positive")
			}

			amountToSend := baseAmount

			var trxFee decimal.Decimal
			if tc.assetIdentifier == tron.TrxAssetIdentifier {
				fee, err := tr.EstimateActivationFee(ctx, tc.fromAddress, tc.toAddress)
				require.NoError(t, err)

				trxFee = fee.Trx

				amountToSend = amountToSend.Sub(fee.Trx)
			}

			require.Equal(t, tc.expectedFee.String(), trxFee.String())

			fmt.Println("base amount", baseAmount)
			fmt.Println("amount to send", amountToSend)

			res, err := tr.EstimateTransferResources(ctx, tc.fromAddress, tc.toAddress, tc.assetIdentifier, amountToSend, 6)
			require.NoError(t, err)

			require.NotNil(t, res)

			fmt.Printf("res: %+v\n", res)
		})
	}
}

func TestEstimateTransferWithDelegateResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	ownerDelegateAddress := "TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF"
	fromAddress := "TMpjRWKSHT41YveMRJ7QmE6MjwXHyat3Xr"
	toAddress := "TNSgW6fqEJWAptLm7vz2iqA8eGdZ9NeKH3"
	contractAddress := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	res, err := tr.EstimateTransferResources(ctx, fromAddress, toAddress, contractAddress, decimal.NewFromInt(1), 6)
	require.NoError(t, err)

	require.NotNil(t, res)

	fmt.Printf("estimate transfer resources: %+v\n", res)

	// get available resources on processing wallet
	processingResources, err := tr.AvailableForDelegateResources(ctx, ownerDelegateAddress)
	require.NoError(t, err)

	// get available resources on hot wallet
	hotWalletResources, err := tr.TotalAvailableResources(fromAddress)
	require.NoError(t, err)

	// estimate transfer with delegate resources
	res2, err := tr.EstimateTransferWithDelegateResources(ctx, tron.EstimateTransferWithDelegateResourcesRequest{
		ProcessingAddress:   ownerDelegateAddress,
		HotWalletAddress:    fromAddress,
		ProcessingResources: *processingResources,
		HotResources:        *hotWalletResources,
		Estimate:            *res,
	})
	require.NoError(t, err)

	fmt.Printf("estimate transfer with delegate resources: %+v\n", res2)
}

func TestEstimateTransferExternalWithDelegateResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	// ownerDelegateAddress := "TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF"
	fromAddress := "TGw76rGEqFFGhdLbaikAqBSVBjdYTJVgTf" // not-activated hot wallet with trc20
	// toAddress := "TCGPiiHLVrau1fbN7hKGUbDjWGJQyrB2r4"   // activated cold wallet
	toAddress := "TGw76rGEqFFGhdLbaikAqBSVBjdYTJVgTf" // non-activated cold wallet

	contractAddress := "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf"

	estimate, err := tr.EstimateTransferResources(ctx, fromAddress, toAddress, contractAddress, decimal.NewFromInt(1), 6)
	require.NoError(t, err)

	require.NotNil(t, estimate)

	fmt.Printf("estimate transfer resources: %+v\n", estimate)

	// get available resources on hot wallet
	hotWalletResources, err := tr.TotalAvailableResources(fromAddress)
	require.NoError(t, err)

	// estimate transfer with delegate resources
	res2, err := tr.EstimateTransferWithExternalDelegateResources(ctx, tron.EstimateTransferWithExternalDelegateResourcesRequest{
		HotWalletAddress: fromAddress,
		HotResources:     *hotWalletResources,
		Estimate:         *estimate,
	})
	require.NoError(t, err)

	fmt.Printf("estimate transfer with delegate resources: %+v\n", res2)
}

func TestConvertResourceToStackedTRX(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	chainParams, err := tr.ChainParams(ctx)
	require.NoError(t, err)

	accountResources, err := tr.Node().GetAccountResource("TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF")
	require.NoError(t, err)

	coef := decimal.NewFromFloat(1.00005)

	energyTrx := tr.ConvertEnergyToStackedTRX(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, decimal.NewFromFloat(130285).Mul(coef))
	bandwidthTrx := tr.ConvertBandwidthToStackedTRX(accountResources.TotalNetWeight, accountResources.TotalNetLimit, decimal.NewFromFloat(345).Mul(coef))

	fmt.Println("energyTrx", energyTrx, energyTrx.Ceil())
	fmt.Println("bandwidthTrx", bandwidthTrx, bandwidthTrx.Ceil())

	stackedEnergy := tr.ConvertStackedTRXToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, energyTrx.Ceil().IntPart())
	stackedBandwidth := tr.ConvertStackedTRXToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, bandwidthTrx.Ceil().IntPart())

	fmt.Println("stackedEnergy", stackedEnergy)
	fmt.Println("stackedBandwidth", stackedBandwidth)
}

func TestAccountResourceInfo(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	info, err := tr.AccountResourceInfo(ctx, "TQ6DkBmxz3Zk7neh8mwmmkfJsVjrE9wwjY")
	require.NoError(t, err)

	fmt.Printf("TotalStackedTRX: %s\n", info.TotalStackedTRX.String())
	fmt.Printf("StackedBandwidth: %s\n", info.StackedBandwidth.String())
	fmt.Printf("StackedEnergy: %s\n", info.StackedEnergy.String())
	fmt.Printf("StackedBandwidthTRX: %s\n", info.StackedBandwidthTRX.String())
	fmt.Printf("StackedEnergyTRX: %s\n", info.StackedEnergyTRX.String())
	fmt.Printf("EnergyAvailableForUse: %s\n", info.EnergyAvailableForUse.String())
	fmt.Printf("BandwidthAvailableForUse: %s\n", info.BandwidthAvailableForUse.String())
	fmt.Printf("TotalEnergy: %s\n", info.TotalEnergy.String())
	fmt.Printf("TotalBandwidth: %s\n", info.TotalBandwidth.String())
	fmt.Printf("TotalUsedEnergy: %s\n", info.TotalUsedEnergy.String())
	fmt.Printf("TotalUsedBandwidth: %s\n", info.TotalUsedBandwidth.String())
}

func TestConvertStackedTRXToResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	chainParams, err := tr.ChainParams(ctx)
	require.NoError(t, err)

	accountResources, err := tr.Node().GetAccountResource("TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF")
	require.NoError(t, err)

	stackedEnergy := tr.ConvertStackedTRXToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, 11_890_000_000)
	stackedBandwidth := tr.ConvertStackedTRXToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, 246_000_000)

	fmt.Println("stackedEnergy", stackedEnergy)
	fmt.Println("stackedBandwidth", stackedBandwidth)
}

func TestStakedResources(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	tests := []struct {
		address string
	}{
		{
			address: "TWZdGQMJkSxhBaMLX8RoTnrg3A8FHAuaCF",
		},
	}

	for _, tc := range tests {
		fmt.Println("check for", tc.address)

		// check StackedResources
		_, resources, err := tr.StakedResources(ctx, tc.address)
		require.NoError(t, err)

		for _, item := range resources {
			fmt.Printf("%+v\n", item)
		}
	}
}

// Test_MultiSendTRC20 - can be used to send TRC20 tokens to multiple addresses (processing hot wallets perhaps), for example to test transfers with cloud delegation
func Test_MultiSendTRC20(t *testing.T) {
	var (
		contractAddress     = "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf" // TRC20 token contract address
		senderAddress       = "TAZU6DYaUd3WkdBft6hYZMeppcZkeSt4V6" // sender address
		senderPrivateKeyHex = ""                                   // private key in hex
	)

	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	addresses := []string{
		"TLuLsf7qMXYMeaJ5wy4BRbhsEC5hXDph5R",
		"TX9L2WpbzkrpmDiLHg5MKsxxAtfxVG8g7F",
		"TLrgcvBpuW2shDG4Wtk1tKtKEdXoe59iKg",
		"TDTccntEoZnoYfHvgCyMgohiNZewLCXSS1",
		"TW9DnX933Pe8SCJRgQo3RGMexU5B8BBd21",
		"TET9qkCitxnnhwyWUK6N2trG67tvrAc1g7",
		"TN85K1RJf6jvnPwbB9C1V2Gjfvx1ghAE6d",
		"TMz7JEMY2phBoAHTzFKC5r1AQ88HfYPrST",
		"TXprNhWci1Sprwvr7xkZbWDdeTVPAUUfHH",
		"TDHKs3uGfHwGiBJKficKXCcVXoqY6CvQ6P",
	}

	privKey, err := crypto.HexToECDSA(senderPrivateKeyHex)
	require.NoError(t, err)

	txs := utils.NewSlice[*api.TransactionExtention]()
	for _, address := range addresses {
		rndAmount := decimal.NewFromInt32((rand.Int31n(100))).Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(6))))
		tx, err := tr.Node().TRC20Send(senderAddress, address, contractAddress, rndAmount.BigInt(), 30_000_000)
		require.NoError(t, err)
		tr.SignTransaction(tx.Transaction, privKey)
		txs.Add(tx)
	}

	for _, tx := range txs.GetAll() {
		_, err := tr.Node().Broadcast(tx.Transaction)
		t.Logf("broadcasted TRC20 tx with hash: %s", hexutil.Encode(tx.Txid))
		require.NoError(t, err)
	}
}

func Test_ActivateAddress(t *testing.T) {
	var (
	// contractAddress     = "TVwfUJiY9g6XP8gYdETJMdmp1Cbgo9FNm7"                               // TRC20 token contract address
	// senderAddress       = "TBRzENGAyzHrLkEhocdn4rNdTePcAXvTYj"                               // sender address
	// senderPrivateKeyHex = "" // private key in hex
	)

	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	estimate, err := tr.EstimateActivationFee(ctx, tron.BlackHoleAddress, "TVqwnfHBpgnJDgxPfR8dcVDCTrtjyuAkUX")
	require.NoError(t, err)
	t.Logf("estimate: %+v", estimate)
	// for range 100 {
	// 	privkey, err := crypto.HexToECDSA(senderPrivateKeyHex)
	// 	require.NoError(t, err)

	// 	key, err := crypto.GenerateKey()
	// 	require.NoError(t, err)

	// 	pubkeyAddress := address.PubkeyToAddress(key.PublicKey).String()

	// 	addr, err := address.Base58ToAddress(pubkeyAddress)
	// 	require.NoError(t, err)

	// 	tx, err := tr.CreateUnsignedActivationTransaction(ctx, senderAddress, addr.String(), false)
	// 	require.NoError(t, err)

	// 	// sign transaction
	// 	err = tr.SignTransaction(tx.Transaction, privkey)
	// 	require.NoError(t, err)

	// 	// broadcast
	// 	_, err = tr.Node().Broadcast(tx.GetTransaction())
	// 	require.NoError(t, err)
	// }
}

// Test_EmergencyReclaim - can be called to reclaim all resources delegated to other parties by this address
func Test_EmergencyReclaim(t *testing.T) {
	var (
		reclaimerAddr = "TBRzENGAyzHrLkEhocdn4rNdTePcAXvTYj"
		reclaimerKey  = ""
	)

	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := testutils.GetContext()

	err = tr.Start(ctx)
	require.NoError(t, err)

	defer tr.Stop(ctx)

	reclaimerPrivKey, err := crypto.HexToECDSA(reclaimerKey)
	require.NoError(t, err)

	resourceList, err := tr.Node().GetDelegatedResourcesV2(reclaimerAddr)
	require.NoError(t, err)

	for _, entry := range resourceList {
		for _, resource := range entry.DelegatedResource {
			var (
				resourceType   core.ResourceCode
				resourceAmount decimal.Decimal
			)

			txs := utils.NewSlice[*api.TransactionExtention]()

			if resource.FrozenBalanceForBandwidth > 0 {
				resourceType = core.ResourceCode_BANDWIDTH
				resourceAmount = decimal.NewFromInt(resource.FrozenBalanceForBandwidth)

				tx, err := tr.Node().UnDelegateResource(address.Address(resource.GetFrom()).String(), address.Address(resource.GetTo()).String(), resourceType, resourceAmount.IntPart())
				require.NoError(t, err)

				err = tr.SignTransaction(tx.Transaction, reclaimerPrivKey)
				require.NoError(t, err)

				txs.Add(tx)
			}

			if resource.FrozenBalanceForEnergy > 0 {
				resourceType = core.ResourceCode_ENERGY
				resourceAmount = decimal.NewFromInt(resource.FrozenBalanceForEnergy)

				tx, err := tr.Node().UnDelegateResource(address.Address(resource.GetFrom()).String(), address.Address(resource.GetTo()).String(), resourceType, resourceAmount.IntPart())
				require.NoError(t, err)

				err = tr.SignTransaction(tx.Transaction, reclaimerPrivKey)
				require.NoError(t, err)

				txs.Add(tx)
			}

			for _, tx := range txs.GetAll() {
				_, err = tr.Node().Broadcast(tx.Transaction)
				require.NoError(t, err)
			}
		}
	}
}
