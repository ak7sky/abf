package grpcserver

import (
	"context"

	"github.com/ak7sky/abf-service/internal/core"
	"github.com/ak7sky/abf-service/internal/core/model"
	api "github.com/ak7sky/abf-service/internal/grpc/api/gen"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serverHandler struct {
	api.UnimplementedRateLimitServiceServer
	rlsrv core.RateLimitService
}

func newHandler(rlsrv core.RateLimitService) *serverHandler {
	return &serverHandler{rlsrv: rlsrv}
}

func (s *serverHandler) CheckLimits(_ context.Context, data *api.AuthData) (*wrappers.BoolValue, error) {
	ok, err := s.rlsrv.Ok(data.GetLogin(), data.GetPswd(), data.GetIp())
	return &wrappers.BoolValue{Value: ok}, errResponse(err)
}

func (s *serverHandler) ResetLimits(_ context.Context, data *api.AuthData) (*empty.Empty, error) {
	err := s.rlsrv.Reset(data.GetLogin(), data.GetIp())
	return &empty.Empty{}, errResponse(err)
}

func (s *serverHandler) AddToBlack(_ context.Context, ip *api.Ip) (*empty.Empty, error) {
	err := s.rlsrv.AddToList(ip.GetAddr(), uint8(ip.GetMaskLen()), model.Black)
	return &empty.Empty{}, errResponse(err)
}

func (s *serverHandler) AddToWhite(_ context.Context, ip *api.Ip) (*empty.Empty, error) {
	err := s.rlsrv.AddToList(ip.GetAddr(), uint8(ip.GetMaskLen()), model.White)
	return &empty.Empty{}, errResponse(err)
}

func (s *serverHandler) RemoveFromBlack(_ context.Context, ip *api.Ip) (*empty.Empty, error) {
	err := s.rlsrv.RemoveFromList(ip.GetAddr(), uint8(ip.GetMaskLen()), model.Black)
	return &empty.Empty{}, errResponse(err)
}

func (s *serverHandler) RemoveFromWhite(_ context.Context, ip *api.Ip) (*empty.Empty, error) {
	err := s.rlsrv.RemoveFromList(ip.GetAddr(), uint8(ip.GetMaskLen()), model.White)
	return &empty.Empty{}, errResponse(err)
}

func errResponse(errSrv error) error {
	if errSrv != nil {
		return status.Error(codes.Internal, errSrv.Error())
	}
	return nil
}
