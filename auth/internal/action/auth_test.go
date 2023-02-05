//go:build unit || all
// +build unit all

package action_test

import (
	"context"
	"testing"

	"auth/internal/action"
	"auth/internal/model"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var (
	username = "test"
	password = "12345"
	hash     = "6b654c3733216f782a3261686d827ccb0eea8a706c4c34a16891f84e7b"
)

// Заглушка для репозитория
type mockAuthRepository struct {
	mock.Mock
}

func (r *mockAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	args := r.Called(user)

	return args.Error(0)
}

func (r *mockAuthRepository) GetUser(ctx context.Context, login, hashpswrd string) (*model.User, error) {
	args := r.Called(login, hashpswrd)

	if args[0] == nil {
		return nil, args.Error(1)
	}

	return args[0].(*model.User), args.Error(1)
}

// Тесты
type authServiceSuite struct {
	suite.Suite
	repo *mockAuthRepository
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(authServiceSuite))
}

func (s *authServiceSuite) SetupTest() {
	s.repo = new(mockAuthRepository)
}

func (s *authServiceSuite) TestCreateUser() {
	user := &model.User{
		Username: username,
		Password: password,
	}
	userWithHash := &model.User{
		Username: username,
		Password: hash,
	}
	s.repo.On("CreateUser", userWithHash).Return(nil)

	auth := action.NewAuthService(s.repo)
	err := auth.CreateUser(context.Background(), user)
	s.NoError(err, "error must be nil")

	s.repo.AssertExpectations(s.T())
}

func (s *authServiceSuite) TestInvalidLogin() {
	s.repo.On("GetUser", username, hash).Return(nil, model.ErrAuthentication)

	auth := action.NewAuthService(s.repo)
	accessToken, refreshToken, err := auth.Login(context.Background(), username, password)
	s.Empty(accessToken, "access token must be empty")
	s.Empty(refreshToken, "refresh token must be empty")
	s.ErrorIs(err, model.ErrAuthentication, "error must be ErrAuthentication")

	s.repo.AssertExpectations(s.T())
}

func (s *authServiceSuite) TestLoginAndValidateTokens() {
	s.repo.On("GetUser", username, hash).Return(&model.User{
		Username: username,
	}, nil)

	auth := action.NewAuthService(s.repo)
	ctx := context.Background()
	accessToken, refreshToken, err := auth.Login(ctx, username, password)
	s.NotEmpty(accessToken, "access token must be not empty")
	s.NotEmpty(refreshToken, "refresh token must be not empty")
	s.NoError(err, "error must be nil")

	user, accessToken, refreshToken, err := auth.ValidateTokens(ctx, accessToken, refreshToken)
	s.Equal(username, user.Username, "invalid username in token")
	s.Empty(accessToken, "access token must be empty")
	s.Empty(refreshToken, "refresh token must be empty")
	s.NoError(err, "error must be nil")

	s.repo.AssertExpectations(s.T())
}

func (s *authServiceSuite) TestLoginAndValidateInvalidAccessToken() {
	s.repo.On("GetUser", username, hash).Return(&model.User{
		Username: username,
	}, nil)

	auth := action.NewAuthService(s.repo)
	ctx := context.Background()
	accessToken, refreshToken, err := auth.Login(ctx, username, password)
	s.NotEmpty(accessToken, "access token must be not empty")
	s.NotEmpty(refreshToken, "refresh token must be not empty")
	s.NoError(err, "error must be nil")

	user, accessToken, refreshToken, err := auth.ValidateTokens(ctx, "invalidAccessToken", refreshToken)
	s.Equal(username, user.Username, "invalid username in token")
	s.NotEmpty(accessToken, "access token must be not empty")
	s.NotEmpty(refreshToken, "refresh token must be not empty")
	s.NoError(err, "error must be nil")

	s.repo.AssertExpectations(s.T())
}

func (s *authServiceSuite) TestLoginAndValidateInvalidTokens() {
	s.repo.On("GetUser", username, hash).Return(&model.User{
		Username: username,
	}, nil)

	auth := action.NewAuthService(s.repo)
	ctx := context.Background()
	accessToken, refreshToken, err := auth.Login(ctx, username, password)
	s.NotEmpty(accessToken, "access token must be not empty")
	s.NotEmpty(refreshToken, "refresh token must be not empty")
	s.NoError(err, "error must be nil")

	user, accessToken, refreshToken, err := auth.ValidateTokens(ctx, "invalidToken", "invalidToken")
	s.Empty(user, "user must be empty")
	s.Empty(accessToken, "access token must be empty")
	s.Empty(refreshToken, "refresh token must be empty")
	s.Error(err, "error must be not nil")

	s.repo.AssertExpectations(s.T())
}
