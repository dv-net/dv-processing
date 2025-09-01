package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	clientv1 "github.com/dv-net/dv-processing/api/processing/client/v1"
	"github.com/dv-net/dv-processing/api/processing/client/v1/clientv1connect"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/clients"
	"github.com/google/uuid"
)

type clientsServer struct {
	bs baseservices.IBaseServices

	clientv1connect.UnimplementedClientServiceHandler
}

func newClientsServer(
	bs baseservices.IBaseServices,
) *clientsServer {
	return &clientsServer{
		bs: bs,
	}
}

func (s clientsServer) Name() string { return "clients-server" }

func (s *clientsServer) RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return clientv1connect.NewClientServiceHandler(s, opts...)
}

// Create - create a new client
func (s *clientsServer) Create(ctx context.Context, request *connect.Request[clientv1.CreateRequest]) (*connect.Response[clientv1.CreateResponse], error) {
	res, err := s.bs.Clients().Create(ctx, clients.CreateClientDTO{
		CallbackURL:    request.Msg.GetCallbackUrl(),
		BackendVersion: request.Msg.GetBackendVersion(),
		BackendAddress: request.Msg.BackendIp,
		BackendDomain:  request.Msg.MerchantDomain,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create client: %w", err))
	}

	return connect.NewResponse(&clientv1.CreateResponse{
		ClientId:       res.Client.ID.String(),
		ClientKey:      res.Client.SecretKey,
		AdminSecretKey: res.AdminSecret,
	}), nil
}

// UpdateCallbackURL - update client callback url
func (s *clientsServer) UpdateCallbackURL(ctx context.Context, request *connect.Request[clientv1.UpdateCallbackURLRequest]) (*connect.Response[clientv1.UpdateCallbackURLResponse], error) {
	cid, err := uuid.Parse(request.Msg.GetClientId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("client id undefined: %w", err))
	}

	if err = s.bs.Clients().ChangeCallbackURL(ctx, cid, request.Msg.GetCallbackUrl()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update client callback url: %w", err))
	}

	return connect.NewResponse(new(clientv1.UpdateCallbackURLResponse)), nil
}

// GetCallbackURL - update client callback url
func (s *clientsServer) GetCallbackURL(ctx context.Context, request *connect.Request[clientv1.GetCallbackURLRequest]) (*connect.Response[clientv1.GetCallbackURLResponse], error) {
	cid, err := uuid.Parse(request.Msg.GetClientId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("client id undefined: %w", err))
	}
	client, err := s.bs.Clients().GetByID(ctx, cid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get client: %w", err))
	}

	return connect.NewResponse(&clientv1.GetCallbackURLResponse{CallbackUrl: client.CallbackUrl}), nil
}
