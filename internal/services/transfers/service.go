package transfers

import (
	"sync"
	"sync/atomic"

	"github.com/dv-net/dv-processing/internal/blockchains"
	"github.com/dv-net/dv-processing/internal/rmanager"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/valid"
	"github.com/dv-net/mx/logger"
	"github.com/go-playground/validator/v10"
)

type Service struct {
	logger logger.Logger
	config *config.Config
	store  store.IStore

	validator *validator.Validate

	isLocked atomic.Bool
	locker   *sync.Cond

	// Services
	walletsSvc *wallets.Service
	eproxySvc  *eproxy.Service
	rmanager   *rmanager.Service

	blockchains *blockchains.Blockchains
}

func New(
	l logger.Logger,
	conf *config.Config,
	st store.IStore,
	walletsSvc *wallets.Service,
	eproxySvc *eproxy.Service,
	blockchains *blockchains.Blockchains,
	rmanager *rmanager.Service,
) *Service {
	svc := &Service{
		logger:      l,
		config:      conf,
		store:       st,
		validator:   valid.New(),
		locker:      sync.NewCond(&sync.Mutex{}),
		walletsSvc:  walletsSvc,
		eproxySvc:   eproxySvc,
		blockchains: blockchains,
		rmanager:    rmanager,
	}
	return svc
}
