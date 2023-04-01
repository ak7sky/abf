package grpcserver

import (
	"context"

	api "github.com/ak7sky/abf-service/internal/grpc/api/gen"
	"github.com/ak7sky/abf-service/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func loggerInterceptor(logger logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		logger.Info("rpc %s started", info.FullMethod)
		defer logger.Info("rpc %s finished", info.FullMethod)
		logger.Debug("request data: %v", req)
		res, err := handler(ctx, req)
		if err != nil {
			logger.Error("error on rpc %s: %v", info.FullMethod, err)
		}
		return res, err
	}
}

func reqValidatorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		switch info.FullMethod {
		case api.RateLimitService_CheckLimits_FullMethodName:
			reqMsg := req.(*api.AuthData)
			if reqMsg.GetLogin() == "" || reqMsg.GetPswd() == "" || reqMsg.GetIp() == 0 {
				return nil, status.Errorf(
					codes.InvalidArgument, "invalid request: missed required fields (login, pswd, ip)",
				)
			}

		case api.RateLimitService_ResetLimits_FullMethodName:
			reqMsg := req.(*api.AuthData)
			if reqMsg.GetLogin() == "" || reqMsg.GetIp() == 0 {
				return nil, status.Errorf(
					codes.InvalidArgument, "invalid request: missed required fields (login, ip)",
				)
			}

		case api.RateLimitService_AddToBlack_FullMethodName,
			api.RateLimitService_AddToWhite_FullMethodName,
			api.RateLimitService_RemoveFromBlack_FullMethodName,
			api.RateLimitService_RemoveFromWhite_FullMethodName:
			reqMsg := req.(*api.Ip)
			if reqMsg.GetAddr() == 0 || reqMsg.MaskLen == 0 {
				return nil, status.Errorf(
					codes.InvalidArgument, "invalid request: missed required fields (addr, maskLen)",
				)
			}
		}
		return handler(ctx, req)
	}
}
