package auth

import (
	"context"
	"fmt"

	"analytics/internal/adaptors/client/authpb"
	"analytics/internal/domain/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type authService struct {
	client authpb.AuthClient
}

// NewAuthService - конструктор
func NewAuthService(client authpb.AuthClient) *authService {
	return &authService{client}
}

// ValidateToken проверяет токен и получает данные о пользователе из токена
func (s *authService) ValidateToken(ctx context.Context, accessToken, refreshToken string) (
	models.User, string, string, error,
) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "ValidateToken")
	defer span.End()

	rq := authpb.ValidateRq{
		Access:  accessToken,
		Refresh: refreshToken,
	}

	res, err := s.client.Validate(ctx, &rq)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
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
