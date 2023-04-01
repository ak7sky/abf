package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ak7sky/abf-service/internal/core"
	api "github.com/ak7sky/abf-service/internal/grpc/api/gen"
	"github.com/ak7sky/abf-service/internal/logger"
	"google.golang.org/grpc"
)

type AppServer struct {
	server          *grpc.Server
	logger          logger.Logger
	errCh           chan error
	shutdownTimeout time.Duration
}

func Start(rlsrv core.RateLimitService, logger logger.Logger) *AppServer {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggerInterceptor(logger),
			reqValidatorInterceptor(),
		),
	)
	api.RegisterRateLimitServiceServer(grpcServer, newHandler(rlsrv))
	appServer := &AppServer{
		server:          grpcServer,
		logger:          logger,
		errCh:           make(chan error, 1),
		shutdownTimeout: time.Second * 10, // todo init from config
	}
	appServer.start()
	return appServer
}

func (appServer *AppServer) start() {
	listener, err := net.Listen("tcp", "localhost:50051") // TODO init from config
	if err != nil {
		appServer.errCh <- err
		return
	}

	appServer.logger.Info(fmt.Sprintf("starting server on %s", listener.Addr().String()))

	go func() {
		appServer.errCh <- appServer.server.Serve(listener)
		close(appServer.errCh)
	}()
}

func (appServer *AppServer) ErrCh() <-chan error {
	return appServer.errCh
}

func (appServer *AppServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), appServer.shutdownTimeout)
	defer cancel()
	return shutdown(ctx, appServer.server)
}

func shutdown(ctx context.Context, server *grpc.Server) error {
	gracefulStopDone := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(gracefulStopDone)
	}()

	select {
	case <-gracefulStopDone:
		return nil
	case <-ctx.Done():
		server.Stop()
		return ctx.Err()
	}
}
