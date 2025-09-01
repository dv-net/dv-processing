package rmanager

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/mx/clients/connectrpc_client"
	"github.com/dv-net/mx/logger"

	"github.com/dv-net/dv-proto/gen/go/manager/health/v1/healthv1connect"
	"github.com/dv-net/dv-proto/gen/go/manager/order/v1/orderv1connect"
)

type Service struct {
	logger       logger.Logger
	conf         *config.Config
	ordersClient orderv1connect.OrderServiceClient
	healthClient healthv1connect.HealthServiceClient
}

func New(ctx context.Context, logger logger.Logger, conf *config.Config) (*Service, error) {
	ordersClient, err := connectrpc_client.New(
		conf.ResourceManager.Connect, logger, orderv1connect.NewOrderServiceClient,
		connectrpc_client.WithName("order-client"),
		connectrpc_client.WithContext(ctx),
		connectrpc_client.WithConnectrpcOpts(),
	)
	if err != nil {
		return nil, fmt.Errorf("create orders client: %w", err)
	}

	healthClient, err := connectrpc_client.New(
		conf.ResourceManager.Connect, logger, healthv1connect.NewHealthServiceClient,
		connectrpc_client.WithName("health-client"),
		connectrpc_client.WithContext(ctx),
		connectrpc_client.WithConnectrpcOpts(),
	)
	if err != nil {
		return nil, fmt.Errorf("create order client: %w", err)
	}

	return &Service{
		conf:         conf,
		logger:       logger,
		ordersClient: ordersClient,
		healthClient: healthClient,
	}, nil
}

func (o *Service) OrdersClient() orderv1connect.OrderServiceClient { return o.ordersClient }

func (o *Service) HealthClient() healthv1connect.HealthServiceClient { return o.healthClient }
