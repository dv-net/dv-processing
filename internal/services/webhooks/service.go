package webhooks

import (
	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/owners"
	"github.com/dv-net/dv-processing/internal/services/transfers"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/mx/logger"

	svc "github.com/dv-net/mx/service"
)

type Service struct {
	logger logger.Logger
	config *config.Config
	store  store.IStore

	webhookServer    svc.IService
	transfersService *transfers.Service
	ownersService    *owners.Service
}

func New(
	log logger.Logger,
	conf *config.Config,
	store store.IStore,
	transfersService *transfers.Service,
	ownersService *owners.Service,
) *Service {
	return &Service{
		logger:           log,
		config:           conf,
		store:            store,
		webhookServer:    newSender(log, conf, store),
		transfersService: transfersService,
		ownersService:    ownersService,
	}
}

func (s *Service) WebhookServer() svc.IService { return s.webhookServer }
