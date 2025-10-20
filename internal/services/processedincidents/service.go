package processedincidents

import "github.com/dv-net/dv-processing/internal/store"

type Service struct {
	store store.IStore
}

func New(st store.IStore) *Service {
	return &Service{
		store: st,
	}
}
