package baseservices

import (
	"context"

	"github.com/dv-net/mx/logger"

	"github.com/dv-net/dv-processing/internal/blockchains"
	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/dispatcher"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/madmin"
	"github.com/dv-net/dv-processing/internal/rmanager"
	"github.com/dv-net/dv-processing/internal/services/clients"
	"github.com/dv-net/dv-processing/internal/services/owners"
	"github.com/dv-net/dv-processing/internal/services/processedblocks"
	"github.com/dv-net/dv-processing/internal/services/system"
	"github.com/dv-net/dv-processing/internal/services/transfers"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/updater"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
)

type IBaseServices interface { //nolint:interfacebloat
	Clients() *clients.Service
	Owners() *owners.Service
	ProcessedBlocks() *processedblocks.Service
	Wallets() *wallets.Service
	System() system.IService
	Webhooks() *webhooks.Service
	EProxy() *eproxy.Service
	Transfers() *transfers.Service
	Blockchains() *blockchains.Blockchains
	BTC() *btc.BTC
	LTC() *ltc.LTC
	Tron() *tron.Tron
	ETH() *evm.EVM
	BinanceSmartChain() *evm.EVM
	BCH() *bch.BCH
	Doge() *doge.Doge
	MAdmin() *madmin.Service
	RManager() *rmanager.Service
	Updater() *updater.Service
}

type service struct {
	clients         *clients.Service
	owners          *owners.Service
	processedBlocks *processedblocks.Service
	system          system.IService
	wallets         *wallets.Service
	webhooks        *webhooks.Service
	eproxy          *eproxy.Service
	blockchains     *blockchains.Blockchains
	transfers       *transfers.Service
	madmin          *madmin.Service
	rmanager        *rmanager.Service
	upd             *updater.Service
}

func New(
	ctx context.Context,
	l logger.ExtendedLogger,
	conf *config.Config,
	st store.IStore,
	explorerProxySvc *eproxy.Service,
	blockchains *blockchains.Blockchains,
	systemSvc system.IService,
	publisher dispatcher.IService,
	rmanager *rmanager.Service,
	walletSDK *walletsdk.SDK,
) (IBaseServices, error) {
	madmin, err := madmin.NewService(ctx, l, conf)
	if err != nil {
		return nil, err
	}
	clientsSvc := clients.New(st, systemSvc, madmin)
	walletsSvc := wallets.New(l, conf, st, publisher, walletSDK)
	ownersSvc := owners.New(conf, st, walletsSvc)
	processedblocksSvc := processedblocks.New(st)
	transfersSvc := transfers.New(l, conf, st, walletsSvc, explorerProxySvc, blockchains, rmanager)
	webhooksSvc := webhooks.New(l, conf, st, transfersSvc, ownersSvc)
	upd, err := updater.NewService(ctx, l, conf)
	if err != nil {
		return nil, err
	}
	return &service{
		clients:         clientsSvc,
		owners:          ownersSvc,
		processedBlocks: processedblocksSvc,
		wallets:         walletsSvc,
		system:          systemSvc,
		webhooks:        webhooksSvc,
		eproxy:          explorerProxySvc,
		transfers:       transfersSvc,
		blockchains:     blockchains,
		madmin:          madmin,
		rmanager:        rmanager,
		upd:             upd,
	}, nil
}

func (s *service) Clients() *clients.Service                 { return s.clients }
func (s *service) Owners() *owners.Service                   { return s.owners }
func (s *service) ProcessedBlocks() *processedblocks.Service { return s.processedBlocks }
func (s *service) Wallets() *wallets.Service                 { return s.wallets }
func (s *service) System() system.IService                   { return s.system }
func (s *service) Webhooks() *webhooks.Service               { return s.webhooks }
func (s *service) Transfers() *transfers.Service             { return s.transfers }
func (s *service) EProxy() *eproxy.Service                   { return s.eproxy }
func (s *service) Blockchains() *blockchains.Blockchains     { return s.blockchains }
func (s *service) BTC() *btc.BTC                             { return s.blockchains.Bitcoin }
func (s *service) LTC() *ltc.LTC                             { return s.blockchains.Litecoin }
func (s *service) BCH() *bch.BCH                             { return s.blockchains.BitcoinCash }
func (s *service) Tron() *tron.Tron                          { return s.blockchains.Tron }
func (s *service) ETH() *evm.EVM                             { return s.blockchains.Ethereum }
func (s *service) BinanceSmartChain() *evm.EVM               { return s.blockchains.BinanceSmartChain }
func (s *service) MAdmin() *madmin.Service                   { return s.madmin }
func (s *service) RManager() *rmanager.Service               { return s.rmanager }
func (s *service) Updater() *updater.Service                 { return s.upd }
func (s *service) Doge() *doge.Doge                          { return s.blockchains.Dogecoin }
