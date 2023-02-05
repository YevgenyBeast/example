//go:build integration
// +build integration

package itests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"analytics/internal/adaptors/client/authpb"
	mock_authpb "analytics/internal/adaptors/client/authpb/mock"
	"analytics/internal/adaptors/grpcserver"
	adap_http "analytics/internal/adaptors/http"
	"analytics/internal/adaptors/postgre"
	"analytics/internal/common"
	"analytics/internal/domain/analytics"
	"analytics/internal/domain/auth"
	"analytics/internal/domain/models"
	. "analytics/itests/testdata"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	dbUser   = "admin"
	dbPass   = "admin"
	dbName   = "test"
	httpPort = "3000"
	grpcPort = "4000"
)

type analyticsHandlerSuite struct {
	suite.Suite

	httpServer       *adap_http.Server
	grpcServer       *grpcserver.ServerGRPC
	logger           *logrus.Entry
	postgreContainer testcontainers.Container
	ctrl             *gomock.Controller
	clientAuth       *mock_authpb.MockAuthClient
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, &analyticsHandlerSuite{})
}

func (s *analyticsHandlerSuite) SetupSuite() {
	logger := common.InitLogger()
	s.logger = logger

	err := common.InitConfig("../")
	s.Require().NoError(err)

	err = sentry.Init(sentry.ClientOptions{
		Dsn: viper.GetString("dsn"),
	})
	s.Require().NoError(err)

	err = common.InitOtel()
	s.Require().NoError(err)

	ctx := context.Background()

	postgreContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:14.4",
			ExposedPorts: []string{"5432"},
			Env: map[string]string{
				"POSTGRES_DB":       dbName,
				"POSTGRES_USER":     dbUser,
				"POSTGRES_PASSWORD": dbPass,
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections"),
			SkipReaper: true,
			AutoRemove: true,
		},
		Started: true,
	})
	s.Require().NoError(err)
	s.postgreContainer = postgreContainer
	time.Sleep(time.Second * 3)

	host, err := postgreContainer.Host(ctx)
	s.Require().NoError(err)
	port, err := postgreContainer.MappedPort(ctx, "5432")
	s.Require().NoError(err)

	conn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		dbUser,
		dbPass,
		host,
		port.Int(),
		dbName,
	)
	db, err := postgre.New(ctx, conn)
	s.Require().NoError(err)

	s.CreateTable(ctx, db)
	s.GenerateTestData(ctx, db)
	analyticsS := analytics.NewAnalyticsService(db)

	ctrl := gomock.NewController(s.T())
	s.ctrl = ctrl
	clientAuth := mock_authpb.NewMockAuthClient(ctrl)
	s.clientAuth = clientAuth
	authS := auth.NewAuthService(clientAuth)

	srv := adap_http.NewServer(httpPort, analyticsS, authS)
	go func() {
		err := srv.Run()
		s.Require().NoError(err)
	}()
	s.httpServer = srv

	grpcServer := grpcserver.NewGRPCServer(db)
	go func() {
		err := grpcServer.Run(grpcPort)
		s.Require().NoError(err)
	}()
	s.grpcServer = grpcServer

	s.T().Log("Suite setup is done")
}

func (s *analyticsHandlerSuite) CreateTable(ctx context.Context, db *postgre.PostgreDatabase) {
	file, err := os.Open("../schema/0001.sql")
	s.NoError(err)
	defer file.Close()

	size, err := file.Stat()
	s.NoError(err)

	reqSQL := make([]byte, size.Size())
	for {
		_, err := file.Read(reqSQL)
		if err == io.EOF {
			break
		}
	}

	_, err = db.Pool.Exec(ctx, string(reqSQL))
	s.Require().NoError(err)
}

func (s *analyticsHandlerSuite) GenerateTestData(ctx context.Context, db *postgre.PostgreDatabase) {
	var id string
	// Генерируем результаты согласования задач
	for i := 0; i < 15; i++ {
		id = uuid.NewV4().String()
		err := db.SetResult(ctx, models.ResultData{
			TaskID: id,
			Result: i%3 != 0,
		})
		s.Require().NoError(err)
	}
	s.T().Log(id)
	// Генерируем временные метки событий в ходе согласования задач
	timestamp := time.Now()
	db.SetTimestamp(ctx, models.TimestampData{
		TaskID:    id,
		EventType: models.TaskTypeEvent,
		Start:     timestamp,
	})
	for i, val := range Approvers {
		durationMail := time.Duration(time.Second)
		durationLink := time.Duration((i + 1) * int(time.Hour))
		timestamp = timestamp.Add(durationMail)
		db.SetTimestamp(ctx, models.TimestampData{
			TaskID:    id,
			Approver:  val,
			EventType: models.ApproveTypeEvent,
			Start:     timestamp,
		})
		timestamp = timestamp.Add(durationLink)
		db.SetTimestamp(ctx, models.TimestampData{
			TaskID:    id,
			Approver:  val,
			EventType: models.ApproveTypeEvent,
			End:       timestamp,
		})
	}
	db.SetTimestamp(ctx, models.TimestampData{
		TaskID:    id,
		EventType: models.TaskTypeEvent,
		End:       timestamp.Add(time.Duration(int(time.Second))),
	})
}

func setCookies(req *http.Request) *http.Request {
	req.AddCookie(&http.Cookie{
		Name:     models.AccessCookie,
		Value:    AccessToken,
		HttpOnly: true,
	})
	req.AddCookie(&http.Cookie{
		Name:     models.RefreshCookie,
		Value:    RefreshToken,
		HttpOnly: true,
	})

	return req
}

func (s *analyticsHandlerSuite) expectedUser(userLogin string) {
	s.clientAuth.EXPECT().Validate(gomock.Any(), &authpb.ValidateRq{
		Access:  AccessToken,
		Refresh: RefreshToken,
	}).Return(&authpb.ValidateRs{
		User: &authpb.User{
			UserLogin: userLogin,
		},
		Access:  "",
		Refresh: "",
	}, nil)
}

func (s *analyticsHandlerSuite) TearDownSuite() {
	ctx := context.Background()
	s.postgreContainer.Terminate(ctx)
	s.ctrl.Finish()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err := s.httpServer.Shutdown(ctx)
		s.Require().NoError(err)
		wg.Done()
	}()
	go func() {
		s.grpcServer.Stop()
		wg.Done()
	}()
	wg.Wait()

	s.T().Log("Suite stop is done")
}

func (s *analyticsHandlerSuite) TestResulReport() {
	// Настройка mock'а для валидации запроса
	s.expectedUser("anyUser")
	s.T().Log("expect user")
	// Формируем запрос на создание отчёта с результатами задачи
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/analytics/results", httpPort), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	// Парсим запрос
	var report models.ResultsReport
	s.NoError(json.NewDecoder(res.Body).Decode(&report))
	res.Body.Close()
	s.Equal(10, report.ApprovedTasks)
	s.Equal(5, report.DeclinedTasks)
}

func (s *analyticsHandlerSuite) TestTimeReport() {
	// Настройка mock'а для валидации запроса
	s.expectedUser("anyUser")
	s.T().Log("expect user")
	// Формируем запрос на создание отчёта с результатами задачи
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/analytics/time", httpPort), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	// Парсим запрос
	var report []models.TimeReport
	s.NoError(json.NewDecoder(res.Body).Decode(&report))
	res.Body.Close()
	s.Equal(1, len(report))
	s.Equal(fmt.Sprint(time.Duration(6*int(time.Hour))), report[0].ApproveTime)
	s.Equal(fmt.Sprint(time.Duration(6*int(time.Hour)+4*int(time.Second))), report[0].TotalTime)
}
