package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	systemv1 "github.com/dv-net/dv-processing/api/processing/system/v1"
	"github.com/dv-net/dv-processing/api/processing/system/v1/systemv1connect"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
)

type systemService struct {
	bs baseservices.IBaseServices

	systemv1connect.UnimplementedSystemServiceHandler
}

func newSystemServer(
	bs baseservices.IBaseServices,
) *systemService {
	return &systemService{
		bs: bs,
	}
}

func (s *systemService) Name() string { return "system-server" }

func (s *systemService) RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return systemv1connect.NewSystemServiceHandler(s, opts...)
}

func (s *systemService) Info(
	ctx context.Context,
	_ *connect.Request[systemv1.InfoRequest],
) (*connect.Response[systemv1.InfoResponse], error) {
	return connect.NewResponse(&systemv1.InfoResponse{
		Version: s.bs.System().SystemVersion(ctx),
		Commit:  s.bs.System().SystemCommit(ctx),
	}), nil
}

func (s *systemService) CheckNewVersion(
	ctx context.Context,
	_ *connect.Request[systemv1.CheckNewVersionRequest],
) (*connect.Response[systemv1.CheckNewVersionResponse], error) {
	resp, err := s.bs.Updater().CheckNewVersion(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&systemv1.CheckNewVersionResponse{
		Name:             resp.Data.Name,
		InstalledVersion: resp.Data.InstalledVersion,
		AvailableVersion: resp.Data.AvailableVersion,
		NeedForUpdate:    resp.Data.NeedForUpdate,
	}), nil
}

func (s *systemService) UpdateToNewVersion(ctx context.Context, _ *connect.Request[systemv1.UpdateToNewVersionRequest]) (*connect.Response[systemv1.UpdateToNewVersionResponse], error) {
	err := s.bs.Updater().UpdateToNewVersion(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&systemv1.UpdateToNewVersionResponse{
		Status: "Success update",
	}), nil
}
