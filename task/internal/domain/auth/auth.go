package auth

import (
	"context"
	"fmt"
	"task/internal/adaptors/client/authpb"
	"task/internal/domain/models"

	"go.opentelemetry.io/otel"
)

type Service struct {
	client authpb.AuthClient
}

// NewAuthService - конструктор
func NewAuthService(client authpb.AuthClient) *Service {
	return &Service{client}
}

// ValidateToken проверяет токен и получает данные о пользователе из токена
func (s *Service) ValidateToken(ctx context.Context, accessToken, refreshToken string) (
	user models.User, access string, refresh string, err error,
) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "ValidateToken")
	defer span.End()

	rq := authpb.ValidateRq{
		Access:  accessToken,
		Refresh: refreshToken,
	}

	res, err := s.client.Validate(ctx, &rq)
	if err != nil {
		return models.User{}, "", "", fmt.Errorf("invalid token: %w", err)
	}

	return models.User{
			UserLogin: res.User.UserLogin,
			Email:     res.User.UserEmail,
		},
		res.Access,
		res.Refresh,
		nil
}
