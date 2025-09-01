package watcher

import (
	"context"

	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"

	subscriberv1 "github.com/dv-net/dv-proto/gen/go/watcher/subscriber/v1"

	"connectrpc.com/connect"
)

func (s *Service) processMempoolStream(
	ctx context.Context,
	stream *connect.ServerStreamForClient[subscriberv1.SubscribeMempoolResponse],
	blockchain wconstants.BlockchainType,
) error {
	defer func() {
		if stream.Err() != nil {
			s.log.Errorw("stream closed with error", "stream_error", stream.Err(), "blockchain", blockchain)
		}

		s.log.Debugw("reconnecting watcher", "blockchain", blockchain)
		if err := stream.Close(); err != nil {
			s.log.Errorw("grpc stream close error", "error", err, "blockchain", blockchain)
		}
	}()

	for stream.Receive() {
		select {
		case <-ctx.Done():
			return nil
		default:
			tx := stream.Msg().GetTransaction()
			if tx == nil {
				s.log.Debugw("ping from watcher", "msg", stream.Msg().GetPing(), "blockchain", blockchain)

				continue
			}

			err := s.processMempoolTxEvents(ctx, blockchain, tx)
			if err != nil {
				s.log.Errorw("process tx events from mempool", "error", err, "blockchain", blockchain)
				return err
			}

			return nil
		}
	}

	return nil
}
