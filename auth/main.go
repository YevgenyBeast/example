package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"auth/internal/action"
	"auth/internal/adaptor"
	"auth/internal/common"
	handler "auth/internal/handler/http"
	"auth/internal/server"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
)

func main() {
	logger := common.InitLogger()
	if err := common.InitConfig("."); err != nil {
		logger.Fatalf("initializing config was failed: %s", err.Error())
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn: viper.GetString("dsn"),
	})
	if err != nil {
		logger.Fatalf("sentry init was failed: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	err = common.InitOtel()
	if err != nil {
		logger.Fatalf("jaeger init was failed: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repos, err := adaptor.NewAuthMongoRepository(ctx, viper.GetString("mongo-conn"), viper.GetString("mongo-db"))
	if err != nil {
		logger.Fatalf("mongoDB init was failed: %s", err)
	}
	di := action.NewContainer(repos)
	handler := handler.NewHandler(*di)
	grpcServer := server.NewGRPCServer(*di)

	logger.Info("starting service")
	srv := new(server.Server)
	go func() {
		err := srv.Run(handler, viper.GetString("port"))
		if err != nil && !errors.Is(http.ErrServerClosed, err) {
			logger.Fatalf("server start was failed: %s", err.Error())
		}
	}()

	srvM := new(server.Server)
	go func() {
		err := srvM.Run(nil, viper.GetString("port-metrics"))
		if err != nil && !errors.Is(http.ErrServerClosed, err) {
			logger.Fatalf("server-metrics start was failed: %s", err.Error())
		}
	}()

	go func() {
		err := grpcServer.Run(viper.GetString("grpc-port"))
		if err != nil {
			logger.Fatalf("grpc-server start was failed: %s", err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c

	logger.Info("stopping service")
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		err = srv.Shutdown(ctx)
		if err != nil {
			logger.Fatalf("server stop was failed: %s", err.Error())
		}
		wg.Done()
	}()
	go func() {
		err = srvM.Shutdown(ctx)
		if err != nil {
			logger.Fatalf("server-metrics stop was failed: %s", err.Error())
		}
		wg.Done()
	}()
	go func() {
		grpcServer.Stop()
		wg.Done()
	}()
	wg.Wait()
}
