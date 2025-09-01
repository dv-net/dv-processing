package madmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dv-net/mx/logger"
	"github.com/go-playground/validator/v10"

	madmin_requests "github.com/dv-net/dv-processing/internal/madmin/requests"
	madmin_responses "github.com/dv-net/dv-processing/internal/madmin/responses"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/valid"
)

const (
	getIPEndpointPath        = "/check-my-ip"
	postRegisterEndpointPath = "/merchant/register"
)

type IMerchantAdminService interface {
	Register(ctx context.Context, request *madmin_requests.RegisterRequest) (*madmin_responses.RegisterResponse, error)
	GetIP(ctx context.Context) (*madmin_responses.GetIPResponse, error)
}

type Service struct {
	logger             logger.Logger
	baseURL            string
	httpClient         *http.Client
	validator          *validator.Validate
	processingIdentity *constants.ProcessingIdentity
}

type Option func(c *Service)

func WithBaseURL(url string) Option {
	return func(c *Service) {
		c.baseURL = url
	}
}

func NewService(ctx context.Context, log logger.Logger, conf *config.Config, opts ...Option) (*Service, error) {
	svc := &Service{
		httpClient: http.DefaultClient,
		logger:     log,
		validator:  valid.New(),
	}

	svc.baseURL = conf.MerchantAdmin.BaseURL

	identity, err := constants.IdentityFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get processing identity: %w", err)
	}
	svc.processingIdentity = &identity

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (o *Service) Register(ctx context.Context, request *madmin_requests.RegisterRequest) (*madmin_responses.RegisterResponse, error) {
	request.ProcessingID = o.processingIdentity.ID
	response := &madmin_responses.RegisterResponse{} // TODO: extract future duplicated parts into a separate function
	if err := o.validateReq(request); err != nil {
		return nil, err
	}
	reqBody, err := utils.StructToBytes(request)
	if err != nil {
		return nil, err
	}
	reqURL, err := url.JoinPath(o.baseURL, postRegisterEndpointPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	res, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseBytes := new(bytes.Buffer)
	if _, err := responseBytes.ReadFrom(res.Body); err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		var errResponse madmin_responses.ErrorResponse
		if err := json.Unmarshal(responseBytes.Bytes(), &errResponse); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error response: %s", errResponse.Message)
	}
	if err := json.Unmarshal(responseBytes.Bytes(), response); err != nil {
		return nil, err
	}
	return response, nil
}

func (o *Service) GetIP(ctx context.Context) (*madmin_responses.GetIPResponse, error) {
	response := &madmin_responses.GetIPResponse{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+getIPEndpointPath, http.NoBody)
	if err != nil {
		return nil, err
	}
	res, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseBytes := new(bytes.Buffer)
	if _, err := responseBytes.ReadFrom(res.Body); err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		var errResponse madmin_responses.ErrorResponse
		if err := json.Unmarshal(responseBytes.Bytes(), &errResponse); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error response: %s", errResponse.Message)
	}
	if err := json.Unmarshal(responseBytes.Bytes(), response); err != nil {
		return nil, err
	}
	return response, nil
}
