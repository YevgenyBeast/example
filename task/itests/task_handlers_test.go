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

	"task/internal/adaptors/analytics"
	mock_analyticspb "task/internal/adaptors/client/analyticspb/mock"
	"task/internal/adaptors/client/authpb"
	mock_authpb "task/internal/adaptors/client/authpb/mock"
	adap_http "task/internal/adaptors/http"
	"task/internal/adaptors/mail"
	"task/internal/adaptors/mongo"
	"task/internal/common"
	"task/internal/domain/auth"
	"task/internal/domain/models"
	"task/internal/domain/task"
	. "task/itests/testdata"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
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

type taskHandlerSuite struct {
	suite.Suite

	httpServer      *adap_http.Server
	logger          *logrus.Entry
	mongoContainer  testcontainers.Container
	ctrl            *gomock.Controller
	clientAuth      *mock_authpb.MockAuthClient
	clientAnalytics *mock_analyticspb.MockAnalyticsClient
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, &taskHandlerSuite{})
}

func (s *taskHandlerSuite) SetupSuite() {
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

	db, err := mongo.New(ctx, conn, dbName)
	s.Require().NoError(err)
	mail := mail.New()

	ctrl := gomock.NewController(s.T())
	s.ctrl = ctrl
	clientAnalytics := mock_analyticspb.NewMockAnalyticsClient(ctrl)
	s.clientAnalytics = clientAnalytics
	analytica := analytics.NewAnalyticsService(clientAnalytics)

	taskS := task.NewTaskService(db, mail, analytica)

	clientAuth := mock_authpb.NewMockAuthClient(ctrl)
	s.clientAuth = clientAuth
	authS := auth.NewAuthService(clientAuth)

	srv := adap_http.NewServer(httpPort, taskS, authS)
	go func() {
		err := srv.Run()
		s.Require().NoError(err)
	}()
	s.httpServer = srv

	s.T().Log("Suite setup is done")
}

func (s *taskHandlerSuite) TearDownSuite() {
	ctx := context.Background()
	s.mongoContainer.Terminate(ctx)
	s.ctrl.Finish()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := s.httpServer.Shutdown(ctx)
		s.Require().NoError(err)
		wg.Done()
	}()
	wg.Wait()

	s.T().Log("Suite stop is done")
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

func (s *taskHandlerSuite) expectedUser(userLogin string) {
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

func (s *taskHandlerSuite) expectedSendTimestamp() {
	s.clientAnalytics.EXPECT().SendTimestamp(gomock.Any(), gomock.Any()).Return(&empty.Empty{}, nil)
}

func (s *taskHandlerSuite) expectedSendResult() {
	s.clientAnalytics.EXPECT().SendResult(gomock.Any(), gomock.Any()).Return(&empty.Empty{}, nil)
}

func (s *taskHandlerSuite) creat(task models.Task) models.TaskRes {
	// Настройка mock'а для валидации запроса
	s.expectedUser(task.InitiatorLogin)
	// Настройка mock'а для отправки данных в analytics
	s.expectedSendTimestamp()
	s.expectedSendTimestamp()
	// Формируем запрос на создание задачи
	jsonTask, err := json.Marshal(task)
	s.NoError(err)
	body := strings.NewReader(string(jsonTask))

	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/tasks/run", httpPort), body)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	// Парсим и проверяем данные из запроса
	var resTask models.TaskRes
	s.NoError(json.NewDecoder(res.Body).Decode(&resTask))
	res.Body.Close()
	s.NotEmpty(resTask.ID)
	s.Equal(task.InitiatorLogin, resTask.InitiatorLogin)
	s.Equal(len(task.ApprovalLogins), len(resTask.Approval))

	return resTask
}

func (s *taskHandlerSuite) getAllTasks() []models.Task {
	// Настройка mock'а для валидации запроса
	s.expectedUser("anyUser")
	// Формируем запрос на получение всех задач
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/tasks/", httpPort), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	// Парсим запрос
	var tasks []models.Task
	s.NoError(json.NewDecoder(res.Body).Decode(&tasks))
	res.Body.Close()

	return tasks
}

func (s *taskHandlerSuite) getTaskByID(id string) models.Task {
	// Настройка mock'а для валидации запроса
	s.expectedUser("anyUser")
	// Формируем запрос на получение задачи по ID
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/tasks/%s", httpPort, id), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос и проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	// Парсим запрос
	var task models.Task
	s.NoError(json.NewDecoder(res.Body).Decode(&task))
	res.Body.Close()

	return task
}

func (s *taskHandlerSuite) delete(id, userLogin string) {
	// Настройка mock'а для валидации запроса
	s.expectedUser(userLogin)
	// Формируем запрос на удаление задачи
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:%s/tasks/%s", httpPort, id), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)
}

func (s *taskHandlerSuite) update(task models.Task, id, initiatorLogin string) models.TaskRes {
	// Настройка mock'а для валидации запроса
	s.expectedUser(initiatorLogin)
	// Формируем запрос на обновление задачи
	jsonTask, err := json.Marshal(task)
	s.NoError(err)
	body := strings.NewReader(string(jsonTask))

	req, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:%s/tasks/%s", httpPort, id), body)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)

	// Парсим и проверяем данные из запроса
	var resTask models.TaskRes
	s.NoError(json.NewDecoder(res.Body).Decode(&resTask))
	res.Body.Close()
	s.Equal(id, resTask.ID)
	s.Equal(initiatorLogin, resTask.InitiatorLogin)
	s.Equal(len(task.ApprovalLogins), len(resTask.Approval))

	return resTask
}

func (s *taskHandlerSuite) approve(id, approvalLogin string, isFinalApprover bool) {
	// Настройка mock'а для валидации запроса
	s.expectedUser(approvalLogin)
	// Настройка mock'а для отправки данных в analytics
	s.expectedSendTimestamp()
	if isFinalApprover {
		s.expectedSendResult()
	}
	s.expectedSendTimestamp()

	// Формируем запрос на согласование задачи
	req, err := http.NewRequest("POST", fmt.Sprintf(
		"http://localhost:%s/tasks/%s/approve/%s",
		httpPort,
		id,
		approvalLogin), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)
}

func (s *taskHandlerSuite) decline(id, approvalLogin string) {
	// Настройка mock'а для валидации запроса
	s.expectedUser(approvalLogin)
	// Настройка mock'а для отправки данных в analytics
	s.expectedSendTimestamp()
	s.expectedSendResult()
	s.expectedSendTimestamp()
	// Формируем запрос на согласование задачи
	req, err := http.NewRequest("POST", fmt.Sprintf(
		"http://localhost:%s/tasks/%s/decline/%s",
		httpPort,
		id,
		approvalLogin), nil)
	s.NoError(err)
	req = setCookies(req)

	// Отправляем запрос, проверяем отсутствие ошибок и код 200
	client := http.Client{}
	res, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, res.StatusCode)
}

func (s *taskHandlerSuite) Test1CreateGetAllGetByIDDeleteTask() {
	// Создаём 2 задачи
	task1 := s.creat(Task1)
	task2 := s.creat(Task2)

	// Получаем все задачи
	tasks := s.getAllTasks()
	s.Equal(2, len(tasks))

	// Удаляем задачу task1
	s.delete(task1.ID, task1.InitiatorLogin)

	// Получаем все задачи
	tasks = s.getAllTasks()
	s.Require().Equal(1, len(tasks))

	// Получаем задачу по ID task2
	task := s.getTaskByID(task2.ID)
	s.Equal(tasks[0].InitiatorLogin, task.InitiatorLogin)
	s.Equal(tasks[0].ApprovalLogins, task.ApprovalLogins)
}

func (s *taskHandlerSuite) Test2CreateDeclineUpdateApprove() {
	// Создаём задачу
	task := s.creat(Task1)

	// Согласование 1-ым участником
	s.approve(task.ID, task.Approval[0].Login, false)

	//Не согласование 2-ым участником
	s.decline(task.ID, task.Approval[1].Login)

	// Получаем задачу и проверяем её статус
	declinedTask := s.getTaskByID(task.ID)
	s.Equal(models.DeclinedTaskStatus, declinedTask.Status)

	//Обновление задачи
	updTask := s.update(Task1Update, task.ID, task.InitiatorLogin)

	// Согласование 1-ым участником
	s.approve(task.ID, updTask.Approval[0].Login, false)

	// Cогласование 2-ым участником
	s.approve(task.ID, updTask.Approval[1].Login, true)

	// Получаем задачу и проверяем её статус
	approvedTask := s.getTaskByID(task.ID)
	s.Equal(models.ApprovedTaskStatus, approvedTask.Status)
}
