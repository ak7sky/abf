package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ak7sky/abf-service/internal/core/service"
	"github.com/ak7sky/abf-service/internal/core/storage/mem"
	grpcserver "github.com/ak7sky/abf-service/internal/grpc/server"
	"github.com/ak7sky/abf-service/internal/logger"
)

func Run() {
	bktStorage := mem.NewBktMemStorage()
	netStorage := mem.NewNetMemStorage()
	// todo bktCapacities init from config
	// todo lvl init from config
	rlsrv := service.NewRateLimitService(netStorage, bktStorage, service.BucketCapacities{})
	appLogger := logger.NewLogger("debug")
	appServer := grpcserver.Start(rlsrv, appLogger)

	// Waiting signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case oss := <-signalCh:
		appLogger.Info(fmt.Sprintf("app stops after receiving a signal %s", oss.String()))
	case err := <-appServer.ErrCh():
		appLogger.Error(fmt.Sprintf("app stops after an err %s", err.Error()))
	}

	// Shutdown
	err := appServer.Shutdown()
	if err != nil {
		appLogger.Error(fmt.Sprintf("app stoped with err %s", err.Error()))
	}
}
