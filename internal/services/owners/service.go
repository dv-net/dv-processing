package owners

import (
	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/valid"
	"github.com/go-playground/validator/v10"
)

type Service struct {
	config     *config.Config
	store      store.IStore
	walletsSvc *wallets.Service

	validator *validator.Validate
}

func New(
	conf *config.Config,
	st store.IStore,
	walletsSvc *wallets.Service,
) *Service {
	return &Service{
		config:     conf,
		store:      st,
		walletsSvc: walletsSvc,
		validator:  valid.New(),
	}
}
