package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/util"
	"github.com/dv-net/dv-processing/pkg/dbutils/pgtypeutils"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/mx/logger"
)

const (
	webhookPollingInterval     = time.Second * 2
	applicationJSONContentType = "application/json"
)

// sender is a service that sends webhooks
type sender struct {
	logger logger.Logger
	config *config.Config
	store  store.IStore

	httpClient *http.Client

	webhookInUse atomic.Bool
}

func newSender(l logger.Logger, conf *config.Config, st store.IStore) *sender {
	return &sender{
		logger:     logger.With(l, "service", "webhook-server"),
		config:     conf,
		store:      st,
		httpClient: &http.Client{Timeout: time.Minute},
	}
}

func (s *sender) Name() string { return "webhook-server" }

func (s *sender) Start(ctx context.Context) error {
	if !s.config.Webhooks.Sender.Enabled {
		s.logger.Warn("webhook sender server is disabled")
		return nil
	}

	ticker := time.NewTicker(webhookPollingInterval)
	defer ticker.Stop()

	// immediately process webhooks after startup service
	go func() {
		if err := s.processWebhooks(ctx); err != nil {
			s.logger.Error(err)
		}
	}()

	// process webhooks by ticker
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			go func() {
				if err := s.processWebhooks(ctx); err != nil {
					s.logger.Error(err)
				}
			}()
		}
	}
}

func (s *sender) Stop(_ context.Context) error { return nil }

func (s *sender) processWebhooks(ctx context.Context) error {
	if !s.webhookInUse.CompareAndSwap(false, true) {
		return nil
	}
	defer s.webhookInUse.Store(false)

	// now := time.Now()
	// s.logger.Debug("start to process unsent webhooks")
	// defer func() {
	// 	s.logger.Debugf("finish processing unsent webhooks in %s", time.Since(now))
	// }()

	if err := s.processAllUnsentWebhooks(ctx); err != nil {
		return fmt.Errorf("processAllUnsentWebhooks error: %w", err)
	}

	return nil
}

func (s *sender) processAllUnsentWebhooks(ctx context.Context) error {
	// get all unsent webhooks
	items, err := s.store.Webhooks().GetUnsent(ctx, models.WebhookStatusNew, s.config.Webhooks.Sender.Quantity)
	if err != nil {
		return fmt.Errorf("get unsent error: %w", err)
	}

	if len(items) == 0 {
		// s.logger.Debug("no unsent webhooks")
		return nil
	}

	sentWebhooks := new(atomic.Int32)

	groupedWebhooks, err := groupByRequestID(items)
	if err != nil {
		return fmt.Errorf("group by request_id error: %w", err)
	}

	now := time.Now()
	for groupName, webhooks := range groupedWebhooks {
		for chunk := range slices.Chunk(webhooks, 100) {
			if err := s.handleChunk(ctx, sentWebhooks, chunk, groupName == "none"); err != nil {
				s.logger.Error(err.Error())
			}
		}
	}

	s.logger.Infow(
		"completed processing unsent webhooks",
		"total", len(items),
		"sent", sentWebhooks.Load(),
		"error", len(items)-int(sentWebhooks.Load()),
		"time", time.Since(now).String(),
	)

	return nil
}

// handleChunk
func (s *sender) handleChunk(ctx context.Context, sentWebhooks *atomic.Int32, chunk []*models.WebhookView, parallel bool) error {
	wg := new(sync.WaitGroup)
	wg.Add(len(chunk))

	for _, item := range chunk {
		fields := []any{
			"id", item.ID.String(),
			"client_id", item.ClientID.String(),
			"callback_url", item.CallbackUrl,
		}

		fn := func() error {
			defer wg.Done()

			// send webhook
			resp, err := s.doRequest(ctx, item.Payload, item.CallbackUrl, item.SecretKey)
			if err != nil {
				whResponseData := resp
				if whResponseData == nil || *whResponseData == "" {
					whResponseData = utils.Pointer(err.Error())
				}

				// increment attempt
				if err := s.store.Webhooks().IncrementAttempt(ctx, item.ID, pgtypeutils.EncodeText(whResponseData)); err != nil {
					return fmt.Errorf("increment attempt for webhook id %s error: %w", item.ID, err)
				}

				return fmt.Errorf("send webhook error: %w", err)
			}

			sentWebhooks.Add(1)
			if s.config.Webhooks.RemoveAfterSent {
				// delete webhook from database
				if err := s.store.Webhooks().DeleteByID(ctx, item.ID); err != nil {
					return fmt.Errorf("delete webhook id %s error: %w", item.ID, err)
				}
			} else {
				// set sent_at to now
				if err := s.store.Webhooks().SetSentAtNow(ctx, item.ID, pgtypeutils.EncodeText(resp)); err != nil {
					return fmt.Errorf("set sent_at for webhook id %s error: %w", item.ID, err)
				}
			}

			// s.logger.Debugw("webhook sent successfully", fields...)

			return nil
		}

		if parallel {
			go func() {
				if err := fn(); err != nil {
					s.logger.Errorw(err.Error(), fields...)
				}
			}()
		} else {
			if err := fn(); err != nil {
				return err
			}
		}
	}

	wg.Wait()

	return nil
}

func (s *sender) doRequest(ctx context.Context, payload []byte, callbackURL, webhookSecretKey string) (*string, error) {
	// validate url
	if _, err := url.ParseRequestURI(callbackURL); err != nil {
		return nil, fmt.Errorf("invalid callback url %s : %w", callbackURL, err)
	}

	// validate secret key
	if webhookSecretKey == "" {
		return nil, fmt.Errorf("empty webhook key")
	}

	payloadBuf := bytes.NewBuffer([]byte{})
	if err := json.Compact(payloadBuf, payload); err != nil {
		return nil, fmt.Errorf("json compact error: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	// define request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, payloadBuf)
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	// set headers
	req.Header = http.Header{
		"Content-Type": {applicationJSONContentType},
		"Accept":       {applicationJSONContentType},
		"X-Sign":       {util.SHA256Signature(payloadBuf.Bytes(), webhookSecretKey)},
	}

	// make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body error: %w", err)
	}

	// check response
	if resp.StatusCode != http.StatusAccepted {
		return utils.Pointer(string(answer)), fmt.Errorf("unexpected response status code: %d, message: %s", resp.StatusCode, string(answer))
	}

	return utils.Pointer(string(answer)), nil
}

// groupByRequestID groups webhooks by request_id
func groupByRequestID(webhooks []*models.WebhookView) (map[string][]*models.WebhookView, error) {
	res := make(map[string][]*models.WebhookView)

	for _, item := range webhooks {
		var payload struct {
			RequestID string `json:"request_id,omitempty"`
		}

		if err := json.Unmarshal(item.Payload, &payload); err != nil {
			return res, fmt.Errorf("unmarshal webhook payload error: %w", err)
		}

		if payload.RequestID == "" || item.Kind != models.WebhookKindTransferStatus {
			payload.RequestID = "none"
		}

		if _, ok := res[payload.RequestID]; !ok {
			res[payload.RequestID] = make([]*models.WebhookView, 0)
		}

		res[payload.RequestID] = append(res[payload.RequestID], item)
	}

	return res, nil
}
