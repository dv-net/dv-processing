package transfers

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/errutils"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	commonv1 "github.com/dv-net/dv-proto/gen/go/manager/common/v1"
	healthv1 "github.com/dv-net/dv-proto/gen/go/manager/health/v1"
	orderv1 "github.com/dv-net/dv-proto/gen/go/manager/order/v1"
	"github.com/dv-net/mx/util"
)

// processTron handle transfer request for tron blockchain
func (s *Service) processTron(ctx context.Context, req *CreateTransferRequest) error {
	if !s.config.Blockchain.Tron.Enabled {
		return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
	}

	// check active transfers with the same from address
	transfersInProgress, err := s.store.Transfers().Find(ctx, FindParams{
		StatusesNotIn: []string{
			constants.TransferStatusCompleted.String(),
			constants.TransferStatusFailed.String(),
		},
		FromAddress: &req.FromAddresses[0],
		Blockchain:  &req.Blockchain,
	})
	if err != nil {
		return fmt.Errorf("find transfers: %w", err)
	}

	if len(transfersInProgress) > 0 {
		return rpccode.GetErrorByCode(rpccode.RPCCodeAddressIsTaken)
	}

	// handle transfer by wallet type
	switch req.walletFromType {
	case constants.WalletTypeProcessing:
		if err := s.processTronProcessingWallets(ctx, req); err != nil {
			return fmt.Errorf("handle processing wallet: %w", err)
		}
	case constants.WalletTypeHot:
		if err := s.processTronHotWallets(ctx, req); err != nil {
			return fmt.Errorf("handle hot wallet: %w", err)
		}
	default:
		return fmt.Errorf("unavailable wallet from type: %s", req.walletFromType)
	}

	return nil
}

func (s *Service) processTronProcessingWallets(ctx context.Context, req *CreateTransferRequest) error {
	// get processing wallet balanceAsset
	balanceAsset, err := s.eproxySvc.AddressBalance(ctx, req.FromAddresses[0], req.AssetIdentifier, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get processing wallet balance: %w", err)
	}

	if !balanceAsset.IsPositive() {
		return fmt.Errorf("%w for transfer. available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), balanceAsset)
	}

	amount := req.Amount.Decimal
	if req.WholeAmount {
		amount = balanceAsset
	}

	if amount.GreaterThan(balanceAsset) {
		return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance), req.Amount.Decimal, balanceAsset)
	}

	assetDecimals, err := s.eproxySvc.AssetDecimals(ctx, wconstants.BlockchainTypeTron, req.AssetIdentifier)
	if err != nil {
		return err
	}

	// estimate transfer resources
	estimate, err := s.blockchains.Tron.EstimateTransferResources(ctx, req.FromAddresses[0], req.ToAddresses[0], req.AssetIdentifier, amount, assetDecimals)
	if err != nil {
		return fmt.Errorf("estimate transfer resources: %w", err)
	}

	req.stateData = map[string]any{
		"handler_current_balance":     balanceAsset,
		"handler_estimated_resources": estimate,
		"handler_transfer_amount":     amount,
	}

	switch constants.TronTransferKind(*req.Kind) {
	case constants.TronTransferKindBurnTRX:
		// get processing wallet balance trx
		balanceTrx, err := s.eproxySvc.AddressBalance(ctx, req.FromAddresses[0], tron.TrxAssetIdentifier, req.Blockchain)
		if err != nil {
			return fmt.Errorf("get processing wallet balance: %w", err)
		}

		if !balanceTrx.IsPositive() {
			return fmt.Errorf("%w for transfer. available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), balanceTrx)
		}

		req.stateData["handler_current_balance_trx"] = balanceTrx

		if req.AssetIdentifier != tron.TrxAssetIdentifier {
			if balanceTrx.LessThan(estimate.Trx) {
				return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance), estimate.Trx, balanceTrx)
			}
		} else {
			needTRXForTransfer := amount.Add(estimate.Trx)
			// If the asset is TRX, we need to check the balance against the estimated TRX
			if balanceTrx.LessThan(needTRXForTransfer) {
				return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance), needTRXForTransfer, balanceTrx)
			}
		}

	case constants.TronTransferKindResources:
		resources, err := s.blockchains.Tron.TotalAvailableResources(req.FromAddresses[0])
		if err != nil {
			return fmt.Errorf("get processing wallet available resources: %w", err)
		}

		req.stateData["handler_available_resources"] = resources

		if resources.Energy.LessThan(estimate.Energy) {
			return fmt.Errorf(
				"processing wallet %w energy for transfer on processing wallet. required: %s, available: %s",
				rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), estimate.Energy, resources.Energy,
			)
		}

		if resources.Bandwidth.LessThan(estimate.Bandwidth) {
			return fmt.Errorf(
				"processing wallet %w bandwidth for transfer on processing wallet. required: %s, available: %s",
				rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), estimate.Bandwidth, resources.Bandwidth,
			)
		}
	case constants.TronTransferKindCloudDelegate:
		if err := s.processCloudDelegate(ctx, req, estimate); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unavailable transfer kind: %s for processing withdrawal", *req.Kind)
	}

	return nil
}

func (s *Service) processTronHotWallets(ctx context.Context, req *CreateTransferRequest) error {
	// get hot wallet balance
	balance, err := s.eproxySvc.AddressBalance(ctx, req.FromAddresses[0], req.AssetIdentifier, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get hot wallet balance: %w", err)
	}

	// check hot wallet balance
	if !balance.IsPositive() {
		return fmt.Errorf("hot wallet %w for transfer. available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), balance)
	}

	amount := req.Amount.Decimal
	if req.WholeAmount {
		amount = balance

		if req.AssetIdentifier == tron.TrxAssetIdentifier {
			activationFee, err := s.blockchains.Tron.EstimateActivationFee(ctx, req.FromAddresses[0], req.ToAddresses[0])
			if err != nil {
				return fmt.Errorf("sub activation fee from amount: %w", err)
			}

			amount = amount.Sub(activationFee.Trx)
		}
	}

	if amount.GreaterThan(balance) {
		return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance), req.Amount.Decimal, balance)
	}

	assetDecimals, err := s.eproxySvc.AssetDecimals(ctx, wconstants.BlockchainTypeTron, req.AssetIdentifier)
	if err != nil {
		return err
	}

	// estimate transfer resources
	estimate, err := s.blockchains.Tron.EstimateTransferResources(ctx, req.FromAddresses[0], req.ToAddresses[0], req.AssetIdentifier, amount, assetDecimals)
	if err != nil {
		return fmt.Errorf("estimate transfer resources: %w", err)
	}

	req.stateData = map[string]any{
		"estimated_resources": estimate,
	}

	switch constants.TronTransferKind(*req.Kind) {
	case constants.TronTransferKindBurnTRX:
		if err := s.processBurnTRX(ctx, req, estimate); err != nil {
			return err
		}
	case constants.TronTransferKindResources:
		if err := s.processResources(ctx, req, estimate); err != nil {
			return err
		}
	case constants.TronTransferKindCloudDelegate:
		if err := s.processCloudDelegate(ctx, req, estimate); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) processBurnTRX(ctx context.Context, req *CreateTransferRequest, estimate *tron.EstimateTransferResourcesResult) error {
	// get processing wallet
	processingWallet, err := s.walletsSvc.Processing().GetByBlockchain(ctx, req.OwnerID, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}

	// get processing wallet trx balance
	processingWalletBalance, err := s.eproxySvc.AddressBalance(ctx, processingWallet.Address, tron.TrxAssetIdentifier, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get processing wallet balance: %w", err)
	}

	activated, err := s.blockchains.Tron.CheckIsWalletActivated(req.FromAddresses[0])
	if err != nil {
		return fmt.Errorf("check wallet activation: %w", err)
	}

	if !activated { //nolint:nestif
		activation := &tron.ActivationResources{}

		// activate from processing wallet with tron system AccountCreateContract
		if s.config.Blockchain.Tron.UseBurnTRXActivation {
			// estimate activation resources using system contract (accountCreate)
			activation, err = s.blockchains.Tron.EstimateSystemContractActivation(ctx, processingWallet.Address, req.FromAddresses[0])
			if err != nil {
				return fmt.Errorf("estimate system activation transaction: %w", err)
			}

			activeTronTransfersBurn, err := s.GetActiveTronTransfersBurn(ctx)
			if err != nil {
				return fmt.Errorf("get active tron transfers burn: %w", err)
			}

			if processingWalletBalance.Sub(activeTronTransfersBurn.ActivationTrx.Add(activeTronTransfersBurn.Trx)).LessThan(estimate.Trx.Add(activation.Trx)) {
				return fmt.Errorf("processing wallet %w for transfer. required: %s TRX, available: %s TRX, active_burn: %s TRX, active_activation: %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), estimate.Trx.Add(activation.Trx), processingWalletBalance, activeTronTransfersBurn, activeTronTransfersBurn.ActivationTrx)
			}

			s.logger.Infow("transfer with system contract activation resources pre-calculation",
				"kind", *req.Kind,
				"estimate_transfer", fmt.Sprintf("%+v", estimate),
				"estimate_activation", fmt.Sprintf("%+v", activation),
			)
		} else {
			processingResources, err := s.blockchains.Tron.AccountResourceInfo(ctx, processingWallet.Address)
			if err != nil {
				return fmt.Errorf("get processing wallet available resources: %w", err)
			}

			activeTronTransfersResources, err := s.GetActiveTronTransfersResources(ctx)
			if err != nil {
				return fmt.Errorf("get active tron transfers: %w", err)
			}

			activeTronTransfersBurn, err := s.GetActiveTronTransfersBurn(ctx)
			if err != nil {
				return fmt.Errorf("get active tron transfers burn: %w", err)
			}

			// estimate activation resources using on-chain contract (activator on-chain contract)
			activation, err = s.blockchains.Tron.EstimateExternalContractActivation(ctx, processingWallet.Address, req.FromAddresses[0])
			if err != nil {
				return fmt.Errorf("estimate activation call: %w", err)
			}

			if processingWalletBalance.Sub(activeTronTransfersBurn.ActivationTrx.Add(activeTronTransfersBurn.Trx)).LessThan(estimate.Trx) {
				return fmt.Errorf("processing wallet %w for transfer. required: %s TRX, available: %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), estimate.Trx, processingWalletBalance)
			}

			if processingResources.TotalAvailableEnergy.Sub(activeTronTransfersResources.Energy.Add(activeTronTransfersResources.ActivationEnergy).Add(activeTronTransfersBurn.ActivationEnergy)).LessThan(activation.Energy) {
				return fmt.Errorf(
					"check active tron transfers with external contract activation. processing wallet %w energy for activation on processing wallet. required: %s, active transfers: %s energy, active activations: %s energy",
					rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), activation.Energy, activeTronTransfersResources.Energy, activeTronTransfersResources.ActivationEnergy.Add(activeTronTransfersBurn.ActivationEnergy),
				)
			}

			if processingResources.TotalAvailableBandwidth.Sub(activeTronTransfersResources.Bandwidth.Add(activeTronTransfersResources.ActivationBandwidth).Add(activeTronTransfersBurn.ActivationBandwidth)).LessThan(activation.Bandwidth) {
				return fmt.Errorf(
					"check active tron transfers with external contract activation. processing wallet %w bandwidth for activation on processing wallet. required: %s, active transfers: %s bandwidth, active activations: %s bandwidth",
					rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), activation.Bandwidth, activeTronTransfersResources.Bandwidth, activeTronTransfersResources.ActivationBandwidth.Add(activeTronTransfersBurn.ActivationBandwidth),
				)
			}

			s.logger.Infow("transfer with external contract activation resources pre-calculation",
				"kind", *req.Kind,
				"estimate_transfer", fmt.Sprintf("%+v", estimate),
				"estimate_activation", fmt.Sprintf("%+v", activation),
			)
		}

		req.stateData = map[string]any{
			"estimated_resources":  estimate,
			"estimated_activation": activation,
		}
	}

	return nil
}

func (s *Service) processResources(ctx context.Context, req *CreateTransferRequest, estimate *tron.EstimateTransferResourcesResult) error {
	// get processing wallet
	processingWallet, err := s.walletsSvc.Processing().GetByBlockchain(ctx, req.OwnerID, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}
	// get available resources on processing wallet
	processingResources, err := s.blockchains.Tron.AvailableForDelegateResources(ctx, processingWallet.Address)
	if err != nil {
		return fmt.Errorf("get processing wallet available resources: %w", err)
	}

	// get available resources on hot wallet
	hotWalletResources, err := s.blockchains.Tron.TotalAvailableResources(req.FromAddresses[0])
	if err != nil {
		return fmt.Errorf("get hot wallet available resources: %w", err)
	}

	// estimate transfer with delegate resources
	res, err := s.blockchains.Tron.EstimateTransferWithDelegateResources(ctx, tron.EstimateTransferWithDelegateResourcesRequest{
		ProcessingAddress:   processingWallet.Address,
		HotWalletAddress:    req.FromAddresses[0],
		ProcessingResources: *processingResources,
		HotResources:        *hotWalletResources,
		Estimate:            *estimate,
	})
	if err != nil {
		return fmt.Errorf("estimate transfer with delegate resources: %w", err)
	}

	// check available bandwidth on processing wallet
	if res.NeedBandwidthFromProcessingWallet.IsPositive() {
		if processingResources.Bandwidth.LessThan(res.NeedBandwidthFromProcessingWallet) {
			return fmt.Errorf("processing wallet %w bandwidth for transfer on processing wallet. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), res.NeedBandwidthFromProcessingWallet, processingResources.Bandwidth)
		}
	}

	// get active tron transfers
	activeTronTransfers, err := s.GetActiveTronTransfersResources(ctx)
	if err != nil {
		return fmt.Errorf("get active tron transfers: %w", err)
	}

	// check transfer energy
	if processingResources.Energy.Sub(activeTronTransfers.Energy).LessThan(res.NeedToDelegate.Energy) {
		return fmt.Errorf(
			"check active tron transfers. processing wallet %w energy for transfer on processing wallet. required: %s, available: %s / active transfers energy: %s",
			rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), res.NeedToDelegate.Energy, processingResources.Energy, activeTronTransfers.Energy,
		)
	}

	// check transfer bandwidth
	if processingResources.Bandwidth.Sub(activeTronTransfers.Bandwidth).LessThan(res.NeedBandwidthFromProcessingWallet) {
		return fmt.Errorf(
			"check active tron transfers. processing wallet %w bandwidth for transfer on processing wallet. required: %s, available: %s / active transfers bandwidth: %s",
			rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), res.NeedBandwidthFromProcessingWallet, processingResources.Bandwidth, activeTronTransfers.Bandwidth,
		)
	}

	activation := &tron.ActivationResources{}

	activated, err := s.blockchains.Tron.CheckIsWalletActivated(req.FromAddresses[0])
	if err != nil {
		return fmt.Errorf("check wallet activation: %w", err)
	}

	if !activated { //nolint:nestif
		processingResources, err := s.blockchains.Tron.AccountResourceInfo(ctx, processingWallet.Address)
		if err != nil {
			return fmt.Errorf("get processing wallet available resources: %w", err)
		}

		if s.config.Blockchain.Tron.UseBurnTRXActivation {
			// estimate system contract activation
			activation, err = s.blockchains.Tron.EstimateSystemContractActivation(ctx, processingWallet.Address, req.FromAddresses[0])
			if err != nil {
				return fmt.Errorf("estimate system activation transaction: %w", err)
			}

			if processingResources.TotalAvailableBandwidth.Sub(activeTronTransfers.Bandwidth.Add(activeTronTransfers.ActivationBandwidth)).LessThan(estimate.Bandwidth.Add(activation.Bandwidth)) {
				return fmt.Errorf("processing wallet %w bandwidth for transfer with activation on processing wallet. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), estimate.Bandwidth.Add(activation.Bandwidth), processingResources.TotalAvailableBandwidth.String())
			}
		} else {
			// estimate activation resources
			activation, err = s.blockchains.Tron.EstimateExternalContractActivation(ctx, processingWallet.Address, req.FromAddresses[0])
			if err != nil {
				return fmt.Errorf("estimate activation call: %w", err)
			}

			if processingResources.TotalAvailableEnergy.Sub(activeTronTransfers.Energy.Add(activeTronTransfers.ActivationEnergy)).LessThan(estimate.Energy.Add(activation.Energy)) {
				return fmt.Errorf(
					"check active tron transfers with system contract activation. processing wallet %w energy for transfer with activation on processing wallet. required: %s, active transfers: %s energy, active activations: %s energy",
					rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), estimate.Energy.Add(activation.Energy), activeTronTransfers.Energy, activeTronTransfers.ActivationEnergy,
				)
			}
			if processingResources.TotalAvailableBandwidth.Sub(activeTronTransfers.Bandwidth.Add(activeTronTransfers.ActivationBandwidth)).LessThan(estimate.Bandwidth.Add(activation.Bandwidth)) {
				return fmt.Errorf(
					"check active tron transfers with activation. processing wallet %w bandwidth for transfer with activation on processing wallet. required: %s, active transfers: %s bandwidth, active activations: %s bandwidth",
					rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources), estimate.Bandwidth.Add(activation.Bandwidth), activeTronTransfers.Bandwidth, activeTronTransfers.ActivationBandwidth,
				)
			}
		}
	}

	req.stateData = map[string]any{
		"estimated_resources":   res,
		"estimated_activation":  activation,
		"active_tron_transfers": activeTronTransfers,
	}

	s.logger.Infow("transfer resources pre-calculation",
		"kind", *req.Kind,
		"estimate_transfer", fmt.Sprintf("%+v", estimate),
		"estimate_activation", fmt.Sprintf("%+v", activation),
		"processing_wallet", fmt.Sprintf("%+v", processingResources),
		"hot_wallet", fmt.Sprintf("%+v", hotWalletResources),
		"need_to_delegate", fmt.Sprintf("%+v", res.NeedToDelegate),
		"active_tron_transfers", fmt.Sprintf("%+v", activeTronTransfers),
		"need_bandwidth_from_processing_wallet", res.NeedBandwidthFromProcessingWallet.String(),
	)

	return nil
}

func (s *Service) processCloudDelegate(ctx context.Context, req *CreateTransferRequest, estimate *tron.EstimateTransferResourcesResult) error {
	// check resource manager health
	{
		eg, egCtx := errgroup.WithContext(ctx)
		eg.Go(func() error { //nolint:dupl
			var res *connect.Response[healthv1.GetHealthStatusResponse]
			var err error

			retryClient := retry.New(
				retry.WithMaxAttempts(2),
				retry.WithPolicy(retry.PolicyLinear),
				retry.WithLogger(s.logger),
			)

			err = retryClient.Do(func() error {
				res, err = s.rmanager.HealthClient().GetHealthStatus(egCtx, connect.NewRequest(&healthv1.GetHealthStatusRequest{
					ServiceType: healthv1.ServiceType_SERVICE_TYPE_ACTIVATOR,
				}))
				// Only retry on network errors
				if err != nil && !errutils.IsNetworkError(err) {
					return retry.ErrExit
				}
				return err
			})
			if err != nil {
				return fmt.Errorf("resource manager health %w", rpccode.GetErrorByCode(rpccode.RPCCodeServiceUnavailable))
			}
			if res.Msg.GetHealthStatus() != healthv1.HealthStatus_HEALTH_STATUS_SERVING {
				return fmt.Errorf("resource manager activator is not serving: %w", rpccode.GetErrorByCode(rpccode.RPCCodeServiceUnavailable))
			}
			return nil
		})
		eg.Go(func() error { //nolint:dupl
			var res *connect.Response[healthv1.GetHealthStatusResponse]
			var err error

			retryClient := retry.New(
				retry.WithMaxAttempts(2),
				retry.WithPolicy(retry.PolicyLinear),
				retry.WithLogger(s.logger),
			)

			err = retryClient.Do(func() error {
				res, err = s.rmanager.HealthClient().GetHealthStatus(egCtx, connect.NewRequest(&healthv1.GetHealthStatusRequest{
					ServiceType: healthv1.ServiceType_SERVICE_TYPE_TRON,
				}))
				// Only retry on network errors
				if err != nil && !errutils.IsNetworkError(err) {
					return retry.ErrExit
				}
				return err
			})
			if err != nil {
				return fmt.Errorf("resource manager health %w", rpccode.GetErrorByCode(rpccode.RPCCodeServiceUnavailable))
			}
			if res.Msg.GetHealthStatus() != healthv1.HealthStatus_HEALTH_STATUS_SERVING {
				return fmt.Errorf("resource manager tron node is not serving: %w", rpccode.GetErrorByCode(rpccode.RPCCodeServiceUnavailable))
			}
			return nil
		})
		if err := eg.Wait(); err != nil {
			return fmt.Errorf("check resource manager health: %w", err)
		}
	}

	hotWalletResources, err := s.blockchains.Tron.TotalAvailableResources(req.FromAddresses[0])
	if err != nil {
		return fmt.Errorf("get hot wallet available resources: %w", err)
	}

	res, err := s.blockchains.Tron.EstimateTransferWithExternalDelegateResources(ctx, tron.EstimateTransferWithExternalDelegateResourcesRequest{
		HotWalletAddress: req.FromAddresses[0],
		HotResources:     *hotWalletResources,
		Estimate:         *estimate,
	})
	if err != nil {
		return fmt.Errorf("estimate transfer with external delegate resources: %w", err)
	}

	req.stateData = map[string]any{
		"estimated_resources": res,
	}

	orders := utils.NewSlice[*orderv1.CreateOrderRequest]()

	// Append activation order if needed
	if res.NeedToActivate {
		orders.Add(&orderv1.CreateOrderRequest{
			OrderType: commonv1.OrderType_ORDER_TYPE_TRON_ADDRESS_ACTIVATION,
			Address:   req.FromAddresses[0],
		})
	}

	// Append bandwidth delegation order if needed
	if !res.NeedResourcesToTransfer.Bandwidth.IsZero() {
		orders.Add(&orderv1.CreateOrderRequest{
			OrderType: commonv1.OrderType_ORDER_TYPE_TRON_BANDWIDTH_DELEGATION,
			Address:   req.FromAddresses[0],
			Duration:  durationpb.New(s.config.ResourceManager.DelegationDuration),
			Amount:    util.Pointer(res.NeedResourcesToTransfer.Bandwidth.IntPart()),
		})
	}

	// Append energy delegation order if needed
	if !res.NeedResourcesToTransfer.Energy.IsZero() {
		orders.Add(&orderv1.CreateOrderRequest{
			OrderType: commonv1.OrderType_ORDER_TYPE_TRON_ENERGY_DELEGATION,
			Address:   req.FromAddresses[0],
			Duration:  durationpb.New(s.config.ResourceManager.DelegationDuration),
			Amount:    util.Pointer(res.NeedResourcesToTransfer.Energy.IntPart()),
		})
	}

	// If there are no orders to create, return early
	if len(orders.GetAll()) != 0 {
		// Submit batch of orders to resource manager
		batchOrders, err := s.rmanager.OrdersClient().BatchCreateOrder(ctx, connect.NewRequest(&orderv1.BatchCreateOrderRequest{
			Orders: orders.GetAll(),
		}))
		if err != nil && connect.CodeOf(err) == connect.CodeResourceExhausted {
			return fmt.Errorf("create orders for transfer: %w", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughResources))
		}
		if err != nil {
			return err
		}
		// Save order IDs to state data
		if batchOrders.Msg.GetOrders() != nil || len(batchOrders.Msg.GetOrders()) != 0 {
			for _, order := range batchOrders.Msg.GetOrders() {
				switch order.OrderType {
				case commonv1.OrderType_ORDER_TYPE_TRON_ADDRESS_ACTIVATION:
					req.stateData["activation_order_id"] = order.OrderId
				case commonv1.OrderType_ORDER_TYPE_TRON_BANDWIDTH_DELEGATION:
					req.stateData["bandwidth_delegation_order_id"] = order.OrderId
				case commonv1.OrderType_ORDER_TYPE_TRON_ENERGY_DELEGATION:
					req.stateData["energy_delegation_order_id"] = order.OrderId
				}
			}
		}
	}

	s.logger.Infow("transfer resources pre-calculation",
		"kind", *req.Kind,
		"estimate_transfer", fmt.Sprintf("%+v", estimate),
		"hot_wallet", fmt.Sprintf("%+v", hotWalletResources),
		"need_resources_to_transfer", fmt.Sprintf("%+v", res.NeedResourcesToTransfer),
		"need_system_resources", fmt.Sprintf("%+v", res.NeedSystemResources),
		"need_activation", res.NeedToActivate,
		"need_resources_to_activate", fmt.Sprintf("%+v", res.NeedResourcesToActivate),
	)
	return nil
}
