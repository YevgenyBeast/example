//go:build unit || all
// +build unit all

package task_test

import (
	"context"
	"strings"
	"testing"

	"task/internal/adaptors/http"
	"task/internal/domain/models"
	"task/internal/domain/task"
	. "task/internal/domain/task/testdata"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Заглушка работы с БД
type mockTaskStorage struct {
	mock.Mock
}

func (ts *mockTaskStorage) InsertTask(ctx context.Context, task models.Task) error {
	task.ID = Task.ID
	args := ts.Called(task)

	return args.Error(0)
}

func (ts *mockTaskStorage) UpdateTask(ctx context.Context, task models.Task) error {
	args := ts.Called(task)

	return args.Error(0)
}

func (ts *mockTaskStorage) DeleteTask(ctx context.Context, id string) error {
	args := ts.Called(id)

	return args.Error(0)
}

func (ts *mockTaskStorage) GetTask(ctx context.Context, id string) (models.Task, error) {
	args := ts.Called(id)

	if args[0] == nil {
		return models.Task{}, args.Error(1)
	}

	return args[0].(models.Task), args.Error(1)
}

func (ts *mockTaskStorage) GetTasks(ctx context.Context) ([]models.Task, error) {
	args := ts.Called()

	if args[0] == nil {
		return nil, args.Error(1)
	}

	return args[0].([]models.Task), args.Error(1)
}

// Заглушка для отправки писем
type mockMail struct {
	mock.Mock
}

func (m *mockMail) SendApprovalMail(ctx context.Context, mail models.MailToApproval) error {
	mail.ApproveLink = replaceID(mail.ApproveLink)
	mail.DeclineLink = replaceID(mail.DeclineLink)
	args := m.Called(mail)

	return args.Error(0)
}

func (m *mockMail) SendResultMail(ctx context.Context, mail models.ResultMail) error {
	args := m.Called(mail)

	return args.Error(0)
}

func replaceID(link string) string {
	arr := strings.Split(link, "/")
	arr[2] = Task.ID
	return strings.Join(arr, "/")
}

// Заглушка для отправки данных в analytics
type mockAnalytics struct {
	mock.Mock
}

func (a *mockAnalytics) SendTimestamp(ctx context.Context, data models.TimestampData) error {
	args := a.Called(gomock.Any())

	return args.Error(0)
}

func (a *mockAnalytics) SendResult(ctx context.Context, data models.ResultData) error {
	args := a.Called(gomock.Any())

	return args.Error(0)
}

// Тесты
type taskServiceSuite struct {
	suite.Suite
	db        *mockTaskStorage
	mail      *mockMail
	analitica *mockAnalytics
}

func TestTaskServiceSuite(t *testing.T) {
	suite.Run(t, new(taskServiceSuite))
}

func (s *taskServiceSuite) SetupTest() {
	s.db = new(mockTaskStorage)
	s.mail = new(mockMail)
	s.analitica = new(mockAnalytics)
}

func (s *taskServiceSuite) TestCreateTask() {
	s.db.On("InsertTask", Task).Return(nil)
	s.mail.On("SendApprovalMail", CreateMail).Return(nil)
	s.analitica.On("SendTimestamp", gomock.Any()).Return(nil)

	ctx := http.ContextWithHost(context.Background(), Host)
	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	createdTask, err := taskService.CreateTask(
		ctx,
		models.Task{
			InitiatorLogin: Task.InitiatorLogin,
			ApprovalLogins: Task.ApprovalLogins,
		},
	)
	s.NoError(err)
	s.Equal(Task.InitiatorLogin, createdTask.InitiatorLogin)
	s.Equal(Task.ApprovalLogins, createdTask.ApprovalLogins)
	s.Equal(Task.CurrentApprovalNumber, createdTask.CurrentApprovalNumber)
	s.Equal(Task.Status, createdTask.Status)

	s.db.AssertExpectations(s.T())
	s.mail.AssertExpectations(s.T())
	s.analitica.AssertExpectations(s.T())
}

func (s *taskServiceSuite) TestUpdateTask() {
	s.db.On("GetTask", OldTask.ID).Return(OldTask, nil)
	s.db.On("UpdateTask", Task).Return(nil)
	s.mail.On("SendResultMail", UpdateMail).Return(nil)
	s.mail.On("SendApprovalMail", CreateMail).Return(nil)

	ctx := http.ContextWithHost(context.Background(), Host)
	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	updatedTask, err := taskService.UpdateTask(
		ctx,
		models.Task{
			ID:             OldTask.ID,
			ApprovalLogins: Task.ApprovalLogins,
		},
		OldTask.InitiatorLogin,
	)
	s.NoError(err)
	s.Equal(Task.InitiatorLogin, updatedTask.InitiatorLogin)
	s.Equal(Task.ApprovalLogins, updatedTask.ApprovalLogins)
	s.Equal(Task.CurrentApprovalNumber, updatedTask.CurrentApprovalNumber)
	s.Equal(Task.Status, updatedTask.Status)

	s.db.AssertExpectations(s.T())
	s.mail.AssertExpectations(s.T())
}

func (s *taskServiceSuite) TestDeleteTask() {
	s.db.On("GetTask", Task.ID).Return(Task, nil)
	s.db.On("DeleteTask", Task.ID).Return(nil)
	s.mail.On("SendResultMail", DeleteMail).Return(nil)

	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	err := taskService.DeleteTask(
		context.Background(),
		Task.ID,
		Task.InitiatorLogin,
	)
	s.NoError(err)

	s.db.AssertExpectations(s.T())
	s.mail.AssertExpectations(s.T())
}

func (s *taskServiceSuite) TestGetTaskByID() {
	s.db.On("GetTask", Task.ID).Return(Task, nil)

	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	receivedTask, err := taskService.GetTaskByID(
		context.Background(),
		Task.ID,
	)
	s.NoError(err)
	s.Equal(Task.InitiatorLogin, receivedTask.InitiatorLogin)
	s.Equal(Task.ApprovalLogins, receivedTask.ApprovalLogins)
	s.Equal(Task.CurrentApprovalNumber, receivedTask.CurrentApprovalNumber)
	s.Equal(Task.Status, receivedTask.Status)

	s.db.AssertExpectations(s.T())
}

func (s *taskServiceSuite) TestGetAllTasks() {
	s.db.On("GetTasks").Return([]models.Task{Task, OldTask}, nil)

	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	receivedTasks, err := taskService.GetAllTasks(context.Background())
	s.NoError(err)
	s.Equal(2, len(receivedTasks))

	s.db.AssertExpectations(s.T())
}

func (s *taskServiceSuite) TestApproveTask() {
	s.db.On("GetTask", Task.ID).Return(Task, nil)
	s.db.On("UpdateTask", TaskApproveStep).Return(nil)
	s.mail.On("SendApprovalMail", ApproveMail).Return(nil)
	s.analitica.On("SendTimestamp", gomock.Any()).Return(nil)

	ctx := http.ContextWithHost(context.Background(), Host)
	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	err := taskService.ApprovalTask(
		ctx,
		Task.ID,
		Task.ApprovalLogins[0],
	)
	s.NoError(err)

	s.db.AssertExpectations(s.T())
	s.mail.AssertExpectations(s.T())
	s.analitica.AssertExpectations(s.T())
}

func (s *taskServiceSuite) TestDeclineTask() {
	s.db.On("GetTask", Task.ID).Return(Task, nil)
	s.db.On("UpdateTask", TaskDeclineStep).Return(nil)
	s.mail.On("SendResultMail", DeclineMail).Return(nil)
	s.analitica.On("SendTimestamp", gomock.Any()).Return(nil)
	s.analitica.On("SendResult", gomock.Any()).Return(nil)

	ctx := http.ContextWithHost(context.Background(), Host)
	taskService := task.NewTaskService(s.db, s.mail, s.analitica)
	err := taskService.DeclineTask(
		ctx,
		Task.ID,
		Task.ApprovalLogins[0],
	)
	s.NoError(err)

	s.db.AssertExpectations(s.T())
	s.mail.AssertExpectations(s.T())
	s.analitica.AssertExpectations(s.T())
}
