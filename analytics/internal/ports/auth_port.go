package ports

import (
	"context"

	"analytics/internal/domain/models"
)

// Auth интерфейс для работы с токенами
type Auth interface {
	ValidateToken(ctx context.Context, accessToken, refreshToken string) (models.User, string, string, error)
}
