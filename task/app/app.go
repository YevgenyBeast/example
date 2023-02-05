package application

import (
	"context"
	"sync"
	"task/internal/adaptors/analytics"
	"task/internal/adaptors/client/analyticspb"
	"task/internal/adaptors/client/authpb"
	"task/internal/adaptors/http"
	mailAdaptor "task/internal/adaptors/mail"
	"task/internal/adaptors/mongo"
	"task/internal/common"
	"task/internal/domain/auth"
	"task/internal/domain/task"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	srv       *http.Server
	srvM      *http.Server
	logger    *logrus.Entry
	analytica dialCtx
	auth      dialCtx
}

type dialCtx struct {
	cancel context.CancelFunc
	conn   *grpc.ClientConn
}

func NewApp(ctx context.Context) *App {
	logger := common.InitLogger()
	if err := common.InitConfig("./"); err != nil {
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

	db, err := mongo.New(ctx, viper.GetString("mongo-conn"), viper.GetString("mongo-db"))
	if err != nil {
		logger.Fatalf("mongoDB init was failed: %s", err)
	}

	mail := mailAdaptor.New()

	cwt1, gCancel1 := context.WithTimeout(ctx, time.Second*5)
	target := viper.GetString("host-analytics") + ":" + viper.GetString("grpc-port")

	connAnalytics, err := grpc.DialContext(cwt1, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		logger.Fatalf("gRPC with analytics init was failed: %s", err)
	}

	clientAnalytics := analyticspb.NewAnalyticsClient(connAnalytics)
	analytica := analytics.NewAnalyticsService(clientAnalytics)

	taskS := task.NewTaskService(db, mail, analytica)

	cwt2, gCancel2 := context.WithTimeout(ctx, time.Second*5)
	target = viper.GetString("host-auth") + ":" + viper.GetString("grpc-port")

	connAuth, err := grpc.DialContext(cwt2, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		logger.Fatalf("gRPC with auth init was failed: %s", err)
	}

	clientAuth := authpb.NewAuthClient(connAuth)
	authS := auth.NewAuthService(clientAuth)

	srv := http.NewServer(viper.GetString("port"), taskS, authS)
	srvM := http.NewServerMetrics(viper.GetString("port-metrics"))

	return &App{
		srv:    srv,
		srvM:   srvM,
		logger: logger,
		analytica: dialCtx{
			cancel: gCancel1,
			conn:   connAnalytics,
		},
		auth: dialCtx{
			cancel: gCancel2,
			conn:   connAuth,
		},
	}
}

func (app *App) Start(ctx context.Context) {
	logger := app.logger

	logger.Info("starting service")

	go func() {
		errSrv := app.srv.Run()
		if errSrv != nil {
			logger.Fatalf("server start was failed: %s", errSrv.Error())
		}
	}()

	go func() {
		errSrv := app.srvM.Run()
		if errSrv != nil {
			logger.Fatalf("server-metrics start was failed: %s", errSrv.Error())
		}
	}()
}

func (app *App) Stop(ctx context.Context) {
	defer app.analytica.cancel()
	defer app.analytica.conn.Close()
	defer app.auth.cancel()
	defer app.auth.conn.Close()

	logger := app.logger

	logger.Info("stopping service")

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		errSrv := app.srv.Shutdown(ctx)
		if errSrv != nil {
			logger.Fatalf("server stop was failed: %s", errSrv.Error())
		}

		wg.Done()
	}()

	go func() {
		errSrv := app.srvM.Shutdown(ctx)
		if errSrv != nil {
			logger.Fatalf("server-metrics stop was failed: %s", errSrv.Error())
		}

		wg.Done()
	}()

	wg.Wait()
}
