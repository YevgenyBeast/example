package action

import (
	"context"

	"auth/internal/model"
)

// Container - контейнер с методами сервисов
type Container struct {
	AuthService
}

// AuthService - сервис авторизации
type AuthService interface {
	Login(ctx context.Context, login, password string) (string, string, error)
	ValidateTokens(ctx context.Context, accessToken, refreshToken string) (*model.User, string, string, error)
	CreateUser(ctx context.Context, user *model.User) error
}

// NewContainer создаёт контейнер с методами
func NewContainer(repos AuthRepository) *Container {
	return &Container{
		AuthService: NewAuthService(repos),
	}
}
