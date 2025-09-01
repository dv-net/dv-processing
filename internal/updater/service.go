package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/updater/requests"
	"github.com/dv-net/dv-processing/internal/updater/responses"
	"github.com/dv-net/dv-processing/pkg/utils"

	"github.com/dv-net/mx/logger"
)

const (
	getNewVersionURL    = "/api/v1/version/dv-processing"
	updateProcessingURL = "/api/v1/update"
)

type IUpdateService interface {
	CheckNewVersion(ctx context.Context) (*responses.GetNewVersionResponse, error)
	Update(ctx context.Context) error
}

type Option func(c *Service)

type Service struct {
	logger     logger.Logger
	baseURL    string
	httpClient *http.Client
}

func NewService(_ context.Context, log logger.Logger, conf *config.Config, opts ...Option) (*Service, error) {
	svc := &Service{
		httpClient: http.DefaultClient,
		logger:     log,
	}

	svc.baseURL = conf.Updater.BaseURL

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (u *Service) CheckNewVersion(ctx context.Context) (*responses.GetNewVersionResponse, error) {
	response := &responses.GetNewVersionResponse{}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.baseURL+getNewVersionURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	res, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseBytes := new(bytes.Buffer)
	if _, err := responseBytes.ReadFrom(res.Body); err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned unexpected status code: %d", res.StatusCode)
	}
	if err := json.Unmarshal(responseBytes.Bytes(), response); err != nil {
		return nil, err
	}
	return response, nil
}

func (u *Service) UpdateToNewVersion(ctx context.Context) error {
	response := &responses.UpdateResponse{}
	request := &requests.UpdateRequest{
		Name: "dv-processing",
	}
	reqBody, err := utils.StructToBytes(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.baseURL+updateProcessingURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	res, err := u.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	responseBytes := new(bytes.Buffer)
	if _, err := responseBytes.ReadFrom(res.Body); err != nil {
		return nil
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unexpected status code: %d", res.StatusCode)
	}
	if err := json.Unmarshal(responseBytes.Bytes(), response); err != nil {
		return err
	}
	return nil
}
