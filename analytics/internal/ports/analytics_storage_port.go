package ports

import (
	"context"
	"time"

	"analytics/internal/domain/models"
)

// AnalyticsStorage интерфейс для работы с БД
type AnalyticsStorage interface {
	GetResults(ctx context.Context) (models.ResultsReport, error)
	SetResult(ctx context.Context, data models.ResultData) error
	GetDuration(ctx context.Context, eventType string) (map[string]time.Duration, error)
	SetTimestamp(ctx context.Context, data models.TimestampData) error
}
