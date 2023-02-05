//go:build integration || all
// +build integration all

package itests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"auth/internal/action"
	"auth/internal/adaptor"
	"auth/internal/common"
	handler "auth/internal/handler/http"
	"auth/internal/model"
	"auth/internal/server"

	"github.com/getsentry/sentry-go"
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

var (
	testUser = model.User{
		Username: "test",
		Password: "qwerty",
		Email:    "test@mail.com",
	}
	accessToken  string
	refreshToken string
)

type handlerSuite struct {
	suite.Suite

	httpServer     *server.Server
	grpcServer     *server.ServerGRPC
	logger         *logrus.Entry
	mongoContainer testcontainers.Container
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, &handlerSuite{})
}

func (s *handlerSuite) SetupSuite() {
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

	// Запуск контейнера mongoDB
	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:4.4",
			ExposedPorts: []string{"27017"},
			Env: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": dbUser,
				"MONGO_INITDB_ROOT_PASSWORD": dbPass,
			},
			WaitingFor: wait.ForLog("Waiting for connections"),
			SkipReaper: true,
			AutoRemove: true,
		},
		Started: true,
	})
	s.Require().NoError(err)
	s.mongoContainer = mongoContainer

	host, err := mongoContainer.Host(ctx)
	s.Require().NoError(err)
	port, err := mongoContainer.MappedPort(ctx, "27017")
	s.Require().NoError(err)

	conn := fmt.Sprintf("mongodb://%s:%s@%s:%d/",
		dbUser,
		dbPass,
		host,
		port.Int())

	repo, err := adaptor.NewAuthMongoRepository(ctx, conn, dbName)
	s.Require().NoError(err)
	di := action.NewContainer(repo)
	handler := handler.NewHandler(*di)
	grpcServer := server.NewGRPCServer(*di)

	var httpServer server.Server
	go func() {
		err := httpServer.Run(handler, httpPort)
		s.Require().ErrorIs(err, http.ErrServerClosed)
	}()

	go func() {
		err := grpcServer.Run(grpcPort)
		s.Require().NoError(err)

	}()
	s.httpServer = &httpServer
	s.grpcServer = grpcServer

	s.T().Log("Suite setup is done")
}

func (s *handlerSuite) TearDownSuite() {
	ctx := context.Background()
	s.mongoContainer.Terminate(ctx)
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

func (s *handlerSuite) TestCreate() {
	// Формируем запрос на создание пользователя
	jsonUser, err := json.Marshal(testUser)
	s.NoError(err)
	body := strings.NewReader(string(jsonUser))

	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/create", httpPort), body)
	s.NoError(err)

	// Отправляем запрос и проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)
}

func (s *handlerSuite) TestLoginAndGetInfo() {
	// Формируем запрос на вход в систему с BasicAuth
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/login", httpPort), nil)
	s.NoError(err)
	req.SetBasicAuth(testUser.Username, testUser.Password)

	// Отправляем запрос, проверяем отсутствие ошибок, код 200 и наличие cookies
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)
	s.Require().Equal(2, len(res.Cookies()))

	// Формируем запрос на проверку информации о пользователе с установкой полученных cookies
	req, err = http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/i", httpPort), nil)
	s.NoError(err)
	req.AddCookie(res.Cookies()[0])
	req.AddCookie(res.Cookies()[1])

	// Сохраняем значение cookies для следующих тестов
	accessToken = res.Cookies()[0].Value
	refreshToken = res.Cookies()[1].Value

	// Отправляем запрос, проверяем отсутствие ошибок, код 200 и информацию о пользователе
	res, err = client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	var user model.User
	s.NoError(json.NewDecoder(res.Body).Decode(&user))
	res.Body.Close()
	s.Equal(testUser.Username, user.Username)
	s.Equal(testUser.Email, user.Email)
}

func (s *handlerSuite) TestLogout() {
	// Формируем запрос на выход из системы с установкой сохранённых ранее cookies
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/logout", httpPort), nil)
	s.NoError(err)
	req.AddCookie(&http.Cookie{
		Name:     model.AccessCookie,
		Value:    accessToken,
		HttpOnly: true,
	})
	req.AddCookie(&http.Cookie{
		Name:     model.RefreshCookie,
		Value:    refreshToken,
		HttpOnly: true,
	})

	// Отправляем запрос, проверяем отсутствие ошибок, код 200 и пустые cookies
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)
	s.Require().Equal(2, len(res.Cookies()))
	s.Empty(res.Cookies()[0].Value)
	s.Empty(res.Cookies()[1].Value)
}

func (s *handlerSuite) TestInvalidGetInfo() {
	// Формируем запрос на проверку информации о пользователе без cookies
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/i", httpPort), nil)
	s.NoError(err)

	// Отправляем запрос, проверяем отсутствие ошибок, код 403
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusForbidden, res.StatusCode)
}
