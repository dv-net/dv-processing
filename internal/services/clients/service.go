package clients

import (
	"github.com/dv-net/dv-processing/internal/madmin"
	"github.com/dv-net/dv-processing/internal/services/system"
	"github.com/dv-net/dv-processing/internal/store"
)

type Service struct {
	store     store.IStore
	systemSvc system.IService
	madminSvc *madmin.Service
}

func New(
	st store.IStore,
	systemSvc system.IService,
	madminSvc *madmin.Service,
) *Service {
	return &Service{
		store:     st,
		madminSvc: madminSvc,
		systemSvc: systemSvc,
	}
}
