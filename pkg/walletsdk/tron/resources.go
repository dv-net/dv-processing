package tron

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/rpccode"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var ResourceCoefficient = decimal.NewFromFloat(1.007)

type AccountResourceInfoData struct {
	EnergyAvailableForUse    decimal.Decimal `json:"energy_available_for_use"`
	BandwidthAvailableForUse decimal.Decimal `json:"bandwidth_available_for_use"`
	TotalEnergy              decimal.Decimal `json:"total_energy"`
	TotalBandwidth           decimal.Decimal `json:"total_bandwidth"`
	TotalStackedTRX          decimal.Decimal `json:"total_stacked_trx"`
	StackedEnergyTRX         decimal.Decimal `json:"stacked_energy_trx"`
	StackedBandwidthTRX      decimal.Decimal `json:"stacked_bandwidth_trx"`
	StackedEnergy            decimal.Decimal `json:"stacked_energy"`
	StackedBandwidth         decimal.Decimal `json:"stacked_bandwidth"`
	TotalUsedEnergy          decimal.Decimal `json:"total_used_energy"`
	TotalUsedBandwidth       decimal.Decimal `json:"total_used_bandwidth"`
	TotalAvailableEnergy     decimal.Decimal `json:"total_available_energy"`
	TotalAvailableBandwidth  decimal.Decimal `json:"total_available_bandwidth"`
}

func (t *Tron) AccountResourceInfo(ctx context.Context, addr string) (*AccountResourceInfoData, error) {
	addrBytes, err := common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	acc, err := t.node.GetAccount(addr)
	if err != nil {
		return nil, err
	}

	accountResources, err := t.node.GetAccountResource(addr)
	if err != nil {
		return nil, err
	}

	ai, err := t.node.Client.GetDelegatedResourceAccountIndexV2(ctx, client.GetMessageBytes(addrBytes))
	if err != nil {
		return nil, err
	}

	delegatedEnergy, delegatedEnergyTrx := decimal.Zero, decimal.Zero
	delegatedBandwidth, delegatedBandwidthTrx := decimal.Zero, decimal.Zero
	delegatedTrx := decimal.Zero
	for _, addrTo := range ai.GetToAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		}
		delegated, err := t.node.Client.GetDelegatedResourceV2(ctx, dm)
		if err != nil {
			return nil, err
		}
		for _, d := range delegated.GetDelegatedResource() {
			if d.GetFrozenBalanceForBandwidth() > 0 {
				delegatedBandwidth = delegatedBandwidth.Add(t.ConvertStackedTRXToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, d.GetFrozenBalanceForBandwidth()))
				delegatedBandwidthTrx = delegatedBandwidthTrx.Add(t.ConvertBandwidthToStackedTRX(accountResources.TotalNetWeight, accountResources.TotalNetLimit, delegatedBandwidth))
				delegatedTrx = delegatedTrx.Add(delegatedBandwidthTrx)
			}
			if d.GetFrozenBalanceForEnergy() > 0 {
				delegatedEnergy = delegatedEnergy.Add(t.ConvertStackedTRXToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, d.GetFrozenBalanceForEnergy()))
				delegatedEnergyTrx = delegatedEnergyTrx.Add(t.ConvertEnergyToStackedTRX(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, delegatedEnergy))
				delegatedTrx = delegatedTrx.Add(delegatedEnergyTrx)
			}
		}
	}

	stackedEnergy, stackedEnergyTrx := decimal.Zero, decimal.Zero
	stackedBandwidth, stackedBandwidthTrx := decimal.Zero, decimal.Zero
	stackedTrx := decimal.Zero
	for _, item := range acc.FrozenV2 {
		if item.Type == core.ResourceCode_BANDWIDTH {
			stackedBandwidth = stackedBandwidth.Add(t.ConvertStackedTRXToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, item.Amount))
			stackedBandwidthTrx = stackedBandwidthTrx.Add(t.ConvertBandwidthToStackedTRX(accountResources.TotalNetWeight, accountResources.TotalNetLimit, stackedBandwidth))
			stackedTrx = stackedTrx.Add(stackedBandwidthTrx)
		}
		if item.Type == core.ResourceCode_ENERGY {
			stackedEnergy = stackedEnergy.Add(t.ConvertStackedTRXToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, item.Amount))
			stackedEnergyTrx = stackedEnergyTrx.Add(t.ConvertEnergyToStackedTRX(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, stackedEnergy))
			stackedTrx = stackedTrx.Add(stackedEnergyTrx)
		}
	}

	resources := &AccountResourceInfoData{
		TotalStackedTRX:          stackedTrx.Add(delegatedTrx).DivRound(decimal.NewFromFloat(1e6), 6),
		StackedBandwidth:         stackedBandwidth.Add(delegatedBandwidth),
		StackedEnergy:            stackedEnergy.Add(delegatedEnergy),
		StackedBandwidthTRX:      stackedBandwidthTrx.Add(delegatedBandwidthTrx).DivRound(decimal.NewFromFloat(1e6), 6),
		StackedEnergyTRX:         stackedEnergyTrx.Add(delegatedEnergyTrx).DivRound(decimal.NewFromFloat(1e6), 6),
		EnergyAvailableForUse:    t.AvailableEnergy(accountResources),
		TotalEnergy:              t.TotalEnergyLimit(accountResources),
		BandwidthAvailableForUse: t.AvailableBandwidth(accountResources),
		TotalBandwidth:           t.TotalBandwidthLimit(accountResources),
		TotalUsedEnergy:          t.TotalEnergyLimit(accountResources).Sub(t.AvailableEnergy(accountResources)),
		TotalUsedBandwidth:       t.TotalBandwidthLimit(accountResources).Sub(t.AvailableBandwidth(accountResources)),
		TotalAvailableEnergy:     t.AvailableEnergy(accountResources),
		TotalAvailableBandwidth:  t.AvailableBandwidth(accountResources),
	}

	if stackedEnergy.LessThan(resources.EnergyAvailableForUse) {
		resources.EnergyAvailableForUse = stackedEnergy
	}

	if stackedBandwidth.LessThan(resources.BandwidthAvailableForUse) {
		resources.BandwidthAvailableForUse = stackedBandwidth
	}

	return resources, nil
}

func (t *Tron) StakedResources(ctx context.Context, addr string) (*api.AccountResourceMessage, []account.FrozenResource, error) {
	addrBytes, err := common.DecodeCheck(addr)
	if err != nil {
		return nil, nil, err
	}

	coreAccount := &core.Account{Address: addrBytes}

	acc, err := t.node.Client.GetAccount(ctx, coreAccount)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(acc.Address, coreAccount.Address) {
		return nil, nil, fmt.Errorf("account not found")
	}

	resource, err := t.node.Client.GetAccountResource(ctx, coreAccount)
	if err != nil {
		return nil, nil, fmt.Errorf("account resource: %w", err)
	}

	// SUM Total freeze V2
	frozenListV2 := make([]account.FrozenResource, 0)

	// Frozen not delegated
	for _, f := range acc.FrozenV2 {
		frozenListV2 = append(frozenListV2, account.FrozenResource{
			Type:   f.GetType(),
			Amount: f.GetAmount(),
		})
	}

	ai, err := t.node.Client.GetDelegatedResourceAccountIndexV2(ctx, client.GetMessageBytes(addrBytes))
	if err != nil {
		return nil, nil, err
	}

	// Fill Delegated V2
	for _, addrTo := range ai.GetToAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		}
		delegated, err := t.node.Client.GetDelegatedResourceV2(ctx, dm)
		if err != nil {
			return nil, nil, err
		}
		for _, d := range delegated.GetDelegatedResource() {
			if d.GetFrozenBalanceForBandwidth() > 0 {
				frozenListV2 = append(frozenListV2, account.FrozenResource{
					Type:       core.ResourceCode_BANDWIDTH,
					Amount:     d.GetFrozenBalanceForBandwidth(),
					Expire:     d.GetExpireTimeForBandwidth(),
					DelegateTo: address.Address(d.GetTo()).String(),
				})
			}
			if d.GetFrozenBalanceForEnergy() > 0 {
				frozenListV2 = append(frozenListV2, account.FrozenResource{
					Type:       core.ResourceCode_ENERGY,
					Amount:     d.GetFrozenBalanceForEnergy(),
					Expire:     d.GetExpireTimeForEnergy(),
					DelegateTo: address.Address(d.GetTo()).String(),
				})
			}
		}
	}

	return resource, frozenListV2, nil
}

// AvailableEnergy calculates the available energy.
func (t *Tron) AvailableEnergy(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.EnergyLimit - res.EnergyUsed)
}

// AvailableBandwidth calculates the available bandwidth.
func (t *Tron) AvailableBandwidth(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.NetLimit + res.GetFreeNetLimit() - res.GetNetUsed() - res.GetFreeNetUsed())
}

func (t *Tron) AvailableBandwidthWithoutFree(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.NetLimit - res.GetNetUsed())
}

func (t *Tron) TotalEnergyLimit(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.EnergyLimit)
}

func (t *Tron) TotalBandwidthLimit(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.NetLimit + res.FreeNetLimit)
}

type ActivationResources struct {
	Energy    decimal.Decimal `json:"energy"`
	Bandwidth decimal.Decimal `json:"bandwidth"`
	Trx       decimal.Decimal `json:"trx"`
}

type Resources struct {
	Energy         decimal.Decimal `json:"energy"`
	Bandwidth      decimal.Decimal `json:"bandwidth"`
	TotalEnergy    decimal.Decimal `json:"total_energy"`
	TotalBandwidth decimal.Decimal `json:"total_bandwidth"`
}

type SystemResources struct {
	NeedForEnergyDelegation    decimal.Decimal `json:"need_for_energy_delegation"`
	NeedForEnergyReclaim       decimal.Decimal `json:"need_for_energy_reclaim"`
	NeedForBandwidthDelegation decimal.Decimal `json:"need_for_bandwidth_delegation"`
	NeedForBandwidthReclaim    decimal.Decimal `json:"need_for_bandwidth_reclaim"`
}

// AvailableForDelegateResources calculates the available energy and bandwidth for delegate to another account.
func (t *Tron) AvailableForDelegateResources(ctx context.Context, addr string) (*Resources, error) {
	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	account, err := t.node.GetAccount(addr)
	if err != nil {
		return nil, err
	}

	accountResources, err := t.node.GetAccountResource(addr)
	if err != nil {
		return nil, err
	}

	stackedEnergy, stackedBandwidth := decimal.Zero, decimal.Zero
	for _, item := range account.FrozenV2 {
		if item.Type == core.ResourceCode_BANDWIDTH {
			stackedBandwidth = stackedBandwidth.Add(t.ConvertStackedTRXToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, item.Amount))
		}
		if item.Type == core.ResourceCode_ENERGY {
			stackedEnergy = stackedEnergy.Add(t.ConvertStackedTRXToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, item.Amount))
		}
	}

	resources := &Resources{
		Energy:         t.AvailableEnergy(accountResources),
		TotalEnergy:    t.TotalEnergyLimit(accountResources),
		Bandwidth:      t.AvailableBandwidth(accountResources),
		TotalBandwidth: t.TotalBandwidthLimit(accountResources),
	}
	if stackedEnergy.LessThan(resources.Energy) {
		resources.Energy = stackedEnergy
	}

	if stackedBandwidth.LessThan(resources.Bandwidth) {
		resources.Bandwidth = stackedBandwidth
	}

	return resources, nil
}

// TotalAvailableResources calculates the total available resources for the account.
func (t *Tron) TotalAvailableResources(addr string) (*Resources, error) {
	accountResources, err := t.node.GetAccountResource(addr)
	if err != nil {
		return nil, err
	}

	resources := &Resources{
		Energy:         t.AvailableEnergy(accountResources),
		Bandwidth:      t.AvailableBandwidth(accountResources),
		TotalEnergy:    t.TotalEnergyLimit(accountResources),
		TotalBandwidth: t.TotalBandwidthLimit(accountResources),
	}

	return resources, nil
}

// ConvertStackedTRXToEnergy converts stacked TRX to energy.
func (t *Tron) ConvertStackedTRXToEnergy(totalEnergyCurrentLimit, totalEnergyWeight, stackedTrx int64) decimal.Decimal {
	return decimal.NewFromInt(stackedTrx).
		Div(decimal.NewFromInt(1e6)).
		Div(decimal.NewFromInt(totalEnergyWeight)).
		Mul(decimal.NewFromInt(totalEnergyCurrentLimit))
}

// ConvertEnergyToStackedTRX converts energy to stacked TRX. Returns value in SUN.
func (t *Tron) ConvertEnergyToStackedTRX(totalEnergyCurrentLimit, totalEnergyWeight int64, energy decimal.Decimal) decimal.Decimal {
	return energy.
		Div(decimal.NewFromInt(totalEnergyCurrentLimit)).
		Mul(decimal.NewFromInt(totalEnergyWeight)).
		Mul(decimal.NewFromInt(1e6))
}

// ConvertStackedTRXToBandwidth converts stacked TRX to bandwidth.
func (t *Tron) ConvertStackedTRXToBandwidth(totalNetWeight, totalNetLimit, stackedTrx int64) decimal.Decimal {
	return decimal.NewFromInt(stackedTrx).
		Div(decimal.NewFromInt(1e6)).
		Div(decimal.NewFromInt(totalNetWeight)).
		Mul(decimal.NewFromInt(totalNetLimit))
}

// ConvertBandwidthToStackedTRX converts bandwidth to stacked TRX. Returns value in SUN.
func (t *Tron) ConvertBandwidthToStackedTRX(totalNetWeight, totalNetLimit int64, bandwidth decimal.Decimal) decimal.Decimal {
	return bandwidth.
		Div(decimal.NewFromInt(totalNetLimit)).
		Mul(decimal.NewFromInt(totalNetWeight)).
		Mul(decimal.NewFromInt(1e6))
}

// GetMaxOperationsCount calculates the maximum operations count based on the available energy and bandwidth.
func (t *Tron) GetMaxOperationsCount(availableEnergy, availableBandwidth int64) uint32 {
	transferEnergyCost := int64(80_000)
	transferBandwidthCost := int64(345)
	resourceBandwidthCost := int64(290)

	availableBandwidthOperations := int(availableBandwidth / (transferBandwidthCost + resourceBandwidthCost*2))
	availableEnergyOperations := int(availableEnergy / transferEnergyCost)

	// return lower value of available energy and bandwidth
	capacity := availableBandwidthOperations
	if availableEnergyOperations < availableBandwidthOperations {
		capacity = availableEnergyOperations
	}

	if capacity < 0 {
		return 0
	}

	return uint32(capacity) //nolint:gosec
}

type EstimateTransferResourcesResult struct {
	Energy    decimal.Decimal `json:"energy"`
	Bandwidth decimal.Decimal `json:"bandwidth"`
	Trx       decimal.Decimal `json:"trx"`
}

// EstimateTransferResources calculates the estimated transfer resources.
func (t *Tron) EstimateTransferResources(
	ctx context.Context, fromAddress, toAddress, contractAddress string, amount decimal.Decimal, decimals int64,
) (*EstimateTransferResourcesResult, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("from address is required")
	}

	if toAddress == "" {
		return nil, fmt.Errorf("to address is required")
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	amount = amount.Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(decimals))))

	var res EstimateTransferResourcesResult

	var tx *core.Transaction
	var data *api.TransactionExtention
	if contractAddress == TrxAssetIdentifier { //nolint:nestif
		err := retry.New(retry.WithMaxAttempts(3), retry.WithContext(ctx)).Do(func() error {
			var err error
			data, err = t.node.Transfer(fromAddress, toAddress, amount.IntPart())
			if err != nil && !strings.Contains(err.Error(), "reset by peer") {
				return fmt.Errorf("transfer: %w", retry.ErrExit)
			}
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("transfer: %w", err)
		}

		tx = data.GetTransaction()
	} else {
		err := retry.New(retry.WithMaxAttempts(3), retry.WithContext(ctx)).Do(func() error {
			var err error
			data, err = t.node.TRC20Send(fromAddress, toAddress, contractAddress, amount.BigInt(), 100*1e6)
			if err != nil && !strings.Contains(err.Error(), "reset by peer") {
				return fmt.Errorf("cannot make tron transaction: %w", retry.ErrExit)
			}
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("cannot make tron transaction: %w", err)
		}

		tx = data.GetTransaction()
	}

	var err error
	res.Bandwidth, err = t.EstimateBandwidth(tx)
	if err != nil {
		return nil, err
	}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	if contractAddress == TrxAssetIdentifier {
		res.Trx = NewBandwidth(res.Bandwidth).ToTRX(chainParams.TransactionFee).ToDecimal()
	} else {
		jsonString := fmt.Sprintf(`[{"address":"%s"},{"uint256":"%s"}]`, toAddress, amount.BigInt())

		err := retry.New(retry.WithMaxAttempts(3), retry.WithContext(ctx)).Do(func() error {
			var err error
			data, err = t.node.TriggerConstantContract(fromAddress, contractAddress, "transfer(address,uint256)", jsonString)
			if err != nil && !strings.Contains(err.Error(), "reset by peer") {
				return fmt.Errorf("cannot trigger contract: %w", retry.ErrExit)
			}
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("cannot make tron transaction: %w", err)
		}

		res.Energy = decimal.NewFromInt(data.EnergyUsed)
		res.Trx = NewEnergy(res.Energy).
			ToTRX(chainParams.EnergyFee).
			ToDecimal().
			Add(
				NewBandwidth(res.Bandwidth).
					ToTRX(chainParams.TransactionFee).
					ToDecimal(),
			)
	}

	return &res, nil
}

// EstimateBandwidth calculates the estimated bandwidth.
func (t *Tron) EstimateBandwidth(tx *core.Transaction) (decimal.Decimal, error) {
	if err := t.fillFakeTX(tx); err != nil {
		return decimal.Decimal{}, err
	}

	return decimal.NewFromInt(int64(proto.Size(tx))).Add(decimal.NewFromInt(64)), nil
}

// EstimateTransferWithDelegateResourcesRequest is the request for EstimateTransferWithDelegateResources.
//
// All fields are required.
type EstimateTransferWithDelegateResourcesRequest struct {
	ProcessingAddress string `json:"processing_address"`
	HotWalletAddress  string `json:"hot_wallet_address"`

	ProcessingResources Resources `json:"processing_resources"`
	HotResources        Resources `json:"hot_resources"`

	Estimate EstimateTransferResourcesResult `json:"estimate"`
}

type EstimateTransferWithDelegateResourcesResponse struct {
	NeedToDelegate                    Resources       `json:"need_to_delegate"`
	NeedBandwidthFromProcessingWallet decimal.Decimal `json:"need_bandwidth_from_processing_wallet"`
}

func (t *Tron) EstimateTransferWithDelegateResources(ctx context.Context, req EstimateTransferWithDelegateResourcesRequest) (*EstimateTransferWithDelegateResourcesResponse, error) {
	if req.ProcessingAddress == "" {
		return nil, fmt.Errorf("processing address is required")
	}

	if req.HotWalletAddress == "" {
		return nil, fmt.Errorf("hot wallet address is required")
	}

	// processing + hot wallet
	totalAvailableEnergy := req.ProcessingResources.Energy.Add(req.HotResources.Energy)
	totalAvailableBandwidth := req.ProcessingResources.Bandwidth
	if req.HotResources.Bandwidth.GreaterThanOrEqual(req.Estimate.Bandwidth) {
		totalAvailableBandwidth = totalAvailableBandwidth.Add(req.HotResources.Bandwidth)
	}

	// add coefficients to energy and bandwidth
	req.Estimate.Energy = req.Estimate.Energy.Mul(ResourceCoefficient).Ceil()
	req.Estimate.Bandwidth = req.Estimate.Bandwidth.Mul(ResourceCoefficient).Ceil()

	// check energy
	if req.Estimate.Energy.GreaterThan(totalAvailableEnergy) {
		return nil, fmt.Errorf("%w energy for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), req.Estimate.Energy, totalAvailableEnergy)
	}

	// check bandwidth
	if req.Estimate.Bandwidth.GreaterThan(totalAvailableBandwidth) {
		return nil, fmt.Errorf("%w bandwidth for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), req.Estimate.Bandwidth, totalAvailableBandwidth)
	}

	needToDelegate := Resources{}
	if req.HotResources.Energy.LessThan(req.Estimate.Energy) {
		needToDelegate.Energy = req.Estimate.Energy.Sub(req.HotResources.Energy)
	}

	if req.HotResources.Bandwidth.LessThan(req.Estimate.Bandwidth) {
		needToDelegate.Bandwidth = req.Estimate.Bandwidth
	}

	needBandwithFromProcessingWallet := needToDelegate.Bandwidth

	allTransactions := []*core.Transaction{}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain params: %w", err)
	}

	// process energy
	if needToDelegate.Energy.IsPositive() {
		energy := NewEnergy(needToDelegate.Energy)

		energySun := energy.ToTRX(chainParams.EnergyFee).ToSUN().IntPart()

		fakeDelegateTx, err := CreateFakeResourceTransaction(req.ProcessingAddress, req.HotWalletAddress, energySun, core.ResourceCode_ENERGY, false)
		if err != nil {
			return nil, fmt.Errorf("fake delegate energy tx: %w", err)
		}

		fakeReclaimTx, err := CreateFakeResourceTransaction(req.ProcessingAddress, req.HotWalletAddress, energySun, core.ResourceCode_ENERGY, true)
		if err != nil {
			return nil, fmt.Errorf("fake reclaim energy tx: %w", err)
		}

		allTransactions = append(allTransactions, fakeDelegateTx, fakeReclaimTx)
	}

	// check is wallet activated
	{
		isActivated, err := t.CheckIsWalletActivated(req.HotWalletAddress)
		if err != nil {
			return nil, fmt.Errorf("check is wallet activated: %w", err)
		}

		if !isActivated {
			if t.conf.UseBurnTRXActivation {
				estimateSystemActivation, err := t.EstimateSystemContractActivation(ctx, req.ProcessingAddress, req.HotWalletAddress)
				if err != nil {
					return nil, fmt.Errorf("estimate system wallet activation: %w", err)
				}

				needBandwithFromProcessingWallet = needBandwithFromProcessingWallet.Add(estimateSystemActivation.Bandwidth)
			}

			// if wallet is not activated, after activation it will have 600 free bandwidth
			if needToDelegate.Bandwidth.IsPositive() && needToDelegate.Bandwidth.LessThan(decimal.NewFromInt(chainParams.FreeNetLimit)) {
				needBandwithFromProcessingWallet = needBandwithFromProcessingWallet.Sub(needToDelegate.Bandwidth)
				needToDelegate.Bandwidth = decimal.Zero
			}
		}
	}

	// process bandwidth
	if needToDelegate.Bandwidth.IsPositive() {
		bandwidth := NewBandwidth(needToDelegate.Bandwidth)

		bandwidthSun := bandwidth.ToTRX(chainParams.TransactionFee).ToSUN().IntPart()

		fakeDelegateTx, err := CreateFakeResourceTransaction(req.ProcessingAddress, req.HotWalletAddress, bandwidthSun, core.ResourceCode_BANDWIDTH, false)
		if err != nil {
			return nil, fmt.Errorf("fake delegate bandwidth tx: %w", err)
		}

		fakeReclaimTx, err := CreateFakeResourceTransaction(req.ProcessingAddress, req.HotWalletAddress, bandwidthSun, core.ResourceCode_BANDWIDTH, true)
		if err != nil {
			return nil, fmt.Errorf("fake reclaim bandwidth tx: %w", err)
		}

		allTransactions = append(allTransactions, fakeDelegateTx, fakeReclaimTx)
	}

	// estimate bandwidth for all transactions
	for _, tx := range allTransactions {
		estimatedBandwidth, err := t.EstimateBandwidth(tx)
		if err != nil {
			return nil, fmt.Errorf("estimate bandwidth: %w", err)
		}

		needBandwithFromProcessingWallet = needBandwithFromProcessingWallet.Add(estimatedBandwidth)
	}

	res := &EstimateTransferWithDelegateResourcesResponse{
		NeedToDelegate:                    needToDelegate,
		NeedBandwidthFromProcessingWallet: needBandwithFromProcessingWallet,
	}

	return res, nil
}

type EstimateTransferWithExternalDelegateResourcesRequest struct {
	HotWalletAddress string `json:"hot_wallet_address"`

	HotResources Resources `json:"hot_resources"`

	Estimate EstimateTransferResourcesResult `json:"estimate"`
}

func (o *EstimateTransferWithExternalDelegateResourcesRequest) Validate() error {
	if o.HotWalletAddress == "" {
		return fmt.Errorf("hot wallet address is required")
	}

	return nil
}

type EstimateTransferWithExternalDelegateResourcesResponse struct {
	NeedResourcesToTransfer Resources           `json:"need_resources_to_transfer"`
	NeedSystemResources     SystemResources     `json:"need_system_resources"`
	NeedResourcesToActivate ActivationResources `json:"need_resources_to_activate"`
	NeedToActivate          bool                `json:"need_to_activate"`
}

func (t *Tron) EstimateTransferWithExternalDelegateResources(ctx context.Context, req EstimateTransferWithExternalDelegateResourcesRequest) (*EstimateTransferWithExternalDelegateResourcesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// add coefficients to energy and bandwidth
	req.Estimate.Energy = req.Estimate.Energy.Mul(ResourceCoefficient).Ceil()
	req.Estimate.Bandwidth = req.Estimate.Bandwidth.Mul(ResourceCoefficient).Ceil()

	needResourceToTransfer := Resources{}
	if req.HotResources.Energy.LessThan(req.Estimate.Energy) {
		needResourceToTransfer.Energy = req.Estimate.Energy.Sub(req.HotResources.Energy)
	}

	if req.HotResources.Bandwidth.LessThan(req.Estimate.Bandwidth) {
		needResourceToTransfer.Bandwidth = req.Estimate.Bandwidth
	}

	needSystemResources := SystemResources{}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain params: %w", err)
	}

	// calculate additional energy required for delegation/reclaim
	if needResourceToTransfer.Energy.IsPositive() {
		energy := NewEnergy(needResourceToTransfer.Energy)

		energySun := energy.ToTRX(chainParams.EnergyFee).ToSUN().IntPart()

		fakeDelegateTx, err := CreateFakeResourceTransaction(BlackHoleAddress, req.HotWalletAddress, energySun, core.ResourceCode_ENERGY, false)
		if err != nil {
			return nil, fmt.Errorf("fake delegate energy tx: %w", err)
		}
		estimatedBandwidth, err := t.EstimateBandwidth(fakeDelegateTx)
		if err != nil {
			return nil, fmt.Errorf("estimate bandwidth: %w", err)
		}
		needSystemResources.NeedForEnergyDelegation = needSystemResources.NeedForEnergyDelegation.Add(estimatedBandwidth)

		fakeReclaimTx, err := CreateFakeResourceTransaction(BlackHoleAddress, req.HotWalletAddress, energySun, core.ResourceCode_ENERGY, true)
		if err != nil {
			return nil, fmt.Errorf("fake reclaim energy tx: %w", err)
		}
		estimatedBandwidth, err = t.EstimateBandwidth(fakeReclaimTx)
		if err != nil {
			return nil, fmt.Errorf("estimate bandwidth: %w", err)
		}
		needSystemResources.NeedForEnergyReclaim = needSystemResources.NeedForEnergyReclaim.Add(estimatedBandwidth)
	}

	needResourcesToActivate := ActivationResources{}
	needToActivate := false

	// check is wallet activated
	{
		isActivated, err := t.CheckIsWalletActivated(req.HotWalletAddress)
		if err != nil {
			return nil, fmt.Errorf("check is wallet activated: %w", err)
		}

		// if wallet is not activated, after activation it will have 600 free bandwidth
		if !isActivated && needResourceToTransfer.Bandwidth.IsPositive() {
			if needResourceToTransfer.Bandwidth.LessThan(decimal.NewFromInt(chainParams.FreeNetLimit)) {
				// needBandwithFromDelegatorWallet = needBandwithFromDelegatorWallet.Sub(needToDelegate.Bandwidth)
				needResourceToTransfer.Bandwidth = decimal.Zero
				activationEstimate, err := t.EstimateExternalContractActivation(ctx, BlackHoleAddress, req.HotWalletAddress)
				if err != nil {
					return nil, fmt.Errorf("estimate activation call: %w", err)
				}
				needResourcesToActivate = *activationEstimate
				needToActivate = true
			}
		}
	}

	// process bandwidth
	if needResourceToTransfer.Bandwidth.IsPositive() {
		bandwidth := NewBandwidth(needResourceToTransfer.Bandwidth)

		bandwidthSun := bandwidth.ToTRX(chainParams.TransactionFee).ToSUN().IntPart()

		fakeDelegateTx, err := CreateFakeResourceTransaction(BlackHoleAddress, req.HotWalletAddress, bandwidthSun, core.ResourceCode_BANDWIDTH, false)
		if err != nil {
			return nil, fmt.Errorf("fake delegate bandwidth tx: %w", err)
		}
		estimatedBandwidth, err := t.EstimateBandwidth(fakeDelegateTx)
		if err != nil {
			return nil, fmt.Errorf("estimate bandwidth: %w", err)
		}
		needSystemResources.NeedForBandwidthDelegation = needSystemResources.NeedForBandwidthDelegation.Add(estimatedBandwidth)

		fakeReclaimTx, err := CreateFakeResourceTransaction(BlackHoleAddress, req.HotWalletAddress, bandwidthSun, core.ResourceCode_BANDWIDTH, true)
		if err != nil {
			return nil, fmt.Errorf("fake reclaim bandwidth tx: %w", err)
		}
		estimatedBandwidth, err = t.EstimateBandwidth(fakeReclaimTx)
		if err != nil {
			return nil, fmt.Errorf("estimate bandwidth: %w", err)
		}
		needSystemResources.NeedForBandwidthReclaim = needSystemResources.NeedForBandwidthReclaim.Add(estimatedBandwidth)
	}

	res := &EstimateTransferWithExternalDelegateResourcesResponse{
		NeedResourcesToTransfer: needResourceToTransfer,
		NeedSystemResources:     needSystemResources,
		NeedResourcesToActivate: needResourcesToActivate,
		NeedToActivate:          needToActivate,
	}

	return res, nil
}

// CreateFakeResourceTransaction creates a fake resource transaction.
func CreateFakeResourceTransaction(fromAddress, toAddress string, amount int64, resourceType core.ResourceCode, reclaim bool) (*core.Transaction, error) {
	addrFromBytes, err := common.DecodeCheck(fromAddress)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := common.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}

	var contract proto.Message
	var transactionContractType core.Transaction_Contract_ContractType

	if !reclaim {
		contract = &core.DelegateResourceContract{
			OwnerAddress:    addrFromBytes,
			ReceiverAddress: addrToBytes,
			Balance:         amount,
			Resource:        resourceType,
			Lock:            false,
			LockPeriod:      0,
		}

		transactionContractType = core.Transaction_Contract_DelegateResourceContract
	} else {
		contract = &core.UnDelegateResourceContract{
			OwnerAddress:    addrFromBytes,
			ReceiverAddress: addrToBytes,
			Balance:         amount,
			Resource:        resourceType,
		}

		transactionContractType = core.Transaction_Contract_UnDelegateResourceContract
	}

	contractAnyType, err := anypb.New(contract)
	if err != nil {
		return nil, err
	}

	refBlockBytes := []byte{0x01, 0x01}
	hash := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	now := time.Now().UnixNano() / int64(time.Millisecond)

	transaction := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: refBlockBytes,
			RefBlockHash:  hash,
			Expiration:    now,
			Timestamp:     now,
			Contract: []*core.Transaction_Contract{
				{
					Type:      transactionContractType,
					Parameter: contractAnyType,
				},
			},
		},
	}

	return transaction, nil
}

func CreateFakeCreateAccountTransaction(fromAddress, toAddress string) (*core.Transaction, error) {
	addrFromBytes, err := common.DecodeCheck(fromAddress)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := common.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}

	refBlockBytes := []byte{0x01, 0x01}
	hash := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	now := time.Now().UnixNano() / int64(time.Millisecond)

	contract := &core.AccountCreateContract{
		OwnerAddress:   addrFromBytes,
		AccountAddress: addrToBytes,
		Type:           core.AccountType_Normal,
	}

	contractAnyType, err := anypb.New(contract)
	if err != nil {
		return nil, err
	}

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: refBlockBytes,
			RefBlockHash:  hash,
			Expiration:    now,
			Timestamp:     now,
			Contract: []*core.Transaction_Contract{
				{
					Type:      core.Transaction_Contract_AccountCreateContract,
					Parameter: contractAnyType,
				},
			},
		},
	}

	return tx, nil
}

// fillFakeTX fills the transaction with fake data.
func (t *Tron) fillFakeTX(tx *core.Transaction) error {
	tx.Ret = nil

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	_, err = h256h.Write(rawData)
	if err != nil {
		return err
	}

	pk, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return err
	}

	signature, err := crypto.Sign(h256h.Sum(nil), pk.ToECDSA())
	if err != nil {
		return err
	}

	tx.Signature = append(tx.Signature, signature)

	return nil
}

func (t *Tron) EstimateExternalContractActivation(ctx context.Context, caller, receiver string) (*ActivationResources, error) {
	tx, err := t.CreateUnsignedActivationTransaction(ctx, caller, receiver, true)
	if err != nil {
		return nil, fmt.Errorf("activation call: %w", err)
	}
	energy := decimal.NewFromInt(tx.EnergyUsed)
	bandwidth, err := t.EstimateBandwidth(tx.Transaction)
	if err != nil {
		return nil, fmt.Errorf("estimate bandwidth failed: %w", err)
	}

	return &ActivationResources{
		Energy:    energy,
		Bandwidth: bandwidth,
	}, nil
}

func (t *Tron) CreateUnsignedActivationTransaction(ctx context.Context, caller string, receiver string, constant bool) (*api.TransactionExtention, error) {
	var err error
	callerAddress, err := address.Base58ToAddress(caller)
	if err != nil {
		return nil, fmt.Errorf("decode caller address: %w", err)
	}
	targetAddress, err := address.Base58ToAddress(receiver)
	if err != nil {
		return nil, fmt.Errorf("decode target address: %w", err)
	}
	contractAddress, err := address.Base58ToAddress(t.conf.ActivationContractAddress)
	if err != nil {
		return nil, fmt.Errorf("decode contract address: %w", err)
	}

	req := ActivatorActivateMethodSignature.String() + "0000000000000000000000000000000000000000000000000000000000000000"[len(targetAddress.Hex())-4:] + targetAddress.Hex()[4:]
	dataBytes, err := common.FromHex(req)
	if err != nil {
		return nil, fmt.Errorf("decode request data: %w", err)
	}

	tsc := &core.TriggerSmartContract{
		OwnerAddress:    callerAddress.Bytes(),
		ContractAddress: contractAddress.Bytes(),
		Data:            dataBytes,
	}

	var tx *api.TransactionExtention
	if constant {
		tx, err = t.Node().Client.TriggerConstantContract(ctx, tsc)
	} else {
		tx, err = t.Node().Client.TriggerContract(ctx, tsc)
	}
	if err != nil {
		return nil, fmt.Errorf("trigger smart contract non-constant method [%s]: %w", "activate", err)
	}
	if tx.Transaction != nil {
		tx.Transaction.RawData.FeeLimit = 30_000_000
		if err := t.Node().UpdateHash(tx); err != nil {
			return nil, fmt.Errorf("update tx hash: %w", err)
		}
	}
	if tx.Result.Code != 0 {
		return nil, fmt.Errorf("failed tx: %s", tx.Result.GetMessage())
	}
	return tx, nil
}

// EstimateSystemActivationResourcesResult
//
// # Bandwidth - amount of bandwidth required for activation contract call
//
// # TRX - amount of TRX required for activation contract call (burning trx for bandwidth and fixed activation fee)
type EstimateSystemActivationResourcesResult struct {
	Bandwidth decimal.Decimal `json:"bandwidth"`
	TRX       decimal.Decimal `json:"trx"`
}

func (t *Tron) EstimateSystemContractActivation(ctx context.Context, caller string, receiver string) (*ActivationResources, error) {
	var err error

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	accountActivationFee := decimal.NewFromInt(chainParams.CreateNewAccountFeeInSystemContract).Div(decimal.NewFromInt(1e6))

	callerAddress, err := address.Base58ToAddress(caller)
	if err != nil {
		return nil, fmt.Errorf("decode caller address: %w", err)
	}
	targetAddress, err := address.Base58ToAddress(receiver)
	if err != nil {
		return nil, fmt.Errorf("decode target address: %w", err)
	}

	acc := &core.AccountCreateContract{
		OwnerAddress:   callerAddress.Bytes(),
		AccountAddress: targetAddress.Bytes(),
		Type:           core.AccountType_Normal,
	}

	var res ActivationResources

	tx, err := t.Node().Client.CreateAccount2(ctx, acc)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		if strings.Contains(string(tx.GetResult().GetMessage()), "Account has existed") {
			return &res, nil
		}
		if strings.Contains(string(tx.GetResult().GetMessage()), "Contract validate error : Validate CreateAccountActuator error, insufficient fee") {
			return nil, fmt.Errorf("%w on processing wallet for activate hot wallet. required: %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance), accountActivationFee.String())
		}
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}

	res.Bandwidth, err = t.EstimateBandwidth(tx.GetTransaction())
	if err != nil {
		return nil, err
	}

	res.Trx = NewBandwidth(res.Bandwidth).ToTRX(chainParams.TransactionFee).ToDecimal().Add(accountActivationFee)

	return &res, nil
}
