package ports

import (
	"context"
	"task/internal/domain/models"
)

// Auth интерфейс для работы с токенами
type Auth interface {
	ValidateToken(ctx context.Context, accessToken, refreshToken string) (models.User, string, string, error)
}
