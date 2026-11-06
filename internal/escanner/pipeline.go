package escanner

import (
	"sync/atomic"

	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

// eventPipeline turns raw blockchain transactions into deposit/transfer
// webhooks. It is shared by the block scanner and the address backfill worker.
type eventPipeline struct {
	logger     logger.Logger
	conf       config.Blockchain
	blockchain wconstants.BlockchainType
	store      store.IStore
	bs         baseservices.IBaseServices
	sdk        *walletsdk.SDK

	client       *river.Client[pgx.Tx] // river client used to enqueue confirmation-check jobs
	parsedHeight *atomic.Int64         // mirrors the scanner cursor; unused for cursor-less commits
}

func newEventPipeline(
	l logger.Logger,
	conf config.Blockchain,
	blockchain wconstants.BlockchainType,
	st store.IStore,
	bs baseservices.IBaseServices,
	sdk *walletsdk.SDK,
	parsedHeight *atomic.Int64,
) *eventPipeline {
	return &eventPipeline{
		logger:       l,
		conf:         conf,
		blockchain:   blockchain,
		store:        st,
		bs:           bs,
		sdk:          sdk,
		parsedHeight: parsedHeight,
	}
}

// SetRiverClient sets the river client used to enqueue confirmation-check jobs.
func (p *eventPipeline) SetRiverClient(client *river.Client[pgx.Tx]) {
	p.client = client
}
