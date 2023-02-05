package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"analytics/internal/adaptors/client/authpb"
	"analytics/internal/adaptors/grpcserver"
	"analytics/internal/adaptors/http"
	"analytics/internal/adaptors/postgre"
	"analytics/internal/common"
	"analytics/internal/domain/analytics"
	"analytics/internal/domain/auth"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	db, err := postgre.New(ctx, viper.GetString("postgre-conn"))
	if err != nil {
		logger.Fatalf("postgreSQL init was failed: %s", err)
	}

	analyticsS := analytics.NewAnalyticsService(db)

	cwt, gCancel := context.WithTimeout(ctx, time.Second*5)
	target := viper.GetString("host-auth") + ":" + "4000" //viper.GetString("grpc-port")
	conn, err := grpc.DialContext(cwt, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		logger.Fatalf("gRPC init was failed: %s", err)
	}
	defer conn.Close()
	defer gCancel()
	client := authpb.NewAuthClient(conn)
	authS := auth.NewAuthService(client)

	srv := http.NewServer(viper.GetString("port"), analyticsS, authS)
	srvM := http.NewServerMetrics(viper.GetString("port-metrics"))
	grpcServer := grpcserver.NewGRPCServer(db)

	logger.Info("starting service")
	go func() {
		err := srv.Run()
		if err != nil {
			logger.Fatalf("server start was failed: %s", err.Error())
		}
	}()

	go func() {
		err := srvM.Run()
		if err != nil {
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
