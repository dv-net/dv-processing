package dispatcher

import (
	"context"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/pkg/event_broker"
)

const serviceName = "dispatcher"

type IService interface {
	CreatedHotWalletDispatcher() *event_broker.Broker[*models.HotWallet]
}

type Service struct {
	createdHotWalletBroker *event_broker.Broker[*models.HotWallet]
}

var _ IService = (*Service)(nil)

func New() *Service {
	return &Service{
		createdHotWalletBroker: event_broker.New[*models.HotWallet](),
	}
}

func (s *Service) CreatedHotWalletDispatcher() *event_broker.Broker[*models.HotWallet] {
	return s.createdHotWalletBroker
}

func (s *Service) Name() string {
	return serviceName
}

func (s *Service) Start(ctx context.Context) error {
	go s.createdHotWalletBroker.Start()

	<-ctx.Done()
	return nil
}

func (s *Service) Stop(_ context.Context) error {
	s.createdHotWalletBroker.Stop()

	return nil
}

func (s *Service) Ping(_ context.Context) error { return nil }
