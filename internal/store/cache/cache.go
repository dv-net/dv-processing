package cache

import (
	"context"
	"time"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/jellydator/ttlcache/v3"
	"github.com/puzpuzpuz/xsync/v4"
)

type ICache interface {
	Clients() *xsync.Map[string, *models.Client]
	Owners() *ttlcache.Cache[string, *models.Owner]
	HotWallets() *xsync.Map[string, *models.HotWallet]
	ProcessingWallets() *xsync.Map[string, *models.ProcessingWallet]
	ColdWallets() *xsync.Map[string, *models.ColdWallet]
	GlobalSettings() *xsync.Map[string, *models.Setting]
}

type cache struct {
	clients           *xsync.Map[string, *models.Client]
	owners            *ttlcache.Cache[string, *models.Owner]
	hotWallets        *xsync.Map[string, *models.HotWallet]
	processingWallets *xsync.Map[string, *models.ProcessingWallet]
	coldWallets       *xsync.Map[string, *models.ColdWallet]
	globalSettings    *xsync.Map[string, *models.Setting]
}

func New() ICache {
	c := &cache{
		clients: xsync.NewMap[string, *models.Client](),
		owners: ttlcache.New(
			ttlcache.WithTTL[string, *models.Owner](time.Hour),
		),
		hotWallets:        xsync.NewMap[string, *models.HotWallet](),
		processingWallets: xsync.NewMap[string, *models.ProcessingWallet](),
		coldWallets:       xsync.NewMap[string, *models.ColdWallet](),
		globalSettings:    xsync.NewMap[string, *models.Setting](),
	}
	return c
}

func (s *cache) Name() string                  { return "internal-cache" }
func (s *cache) Start(_ context.Context) error { return nil }
func (s *cache) Stop(_ context.Context) error  { return nil }

func (s *cache) Clients() *xsync.Map[string, *models.Client]       { return s.clients }
func (s *cache) Owners() *ttlcache.Cache[string, *models.Owner]    { return s.owners }
func (s *cache) HotWallets() *xsync.Map[string, *models.HotWallet] { return s.hotWallets }
func (s *cache) ProcessingWallets() *xsync.Map[string, *models.ProcessingWallet] {
	return s.processingWallets
}
func (s *cache) ColdWallets() *xsync.Map[string, *models.ColdWallet] { return s.coldWallets }
func (s *cache) GlobalSettings() *xsync.Map[string, *models.Setting] { return s.globalSettings }
