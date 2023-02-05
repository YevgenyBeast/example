package ports

import (
	"context"

	"analytics/internal/domain/models"
)

// Analytics интерфейс для работы с отчётами
type Analytics interface {
	CreateResultsReport(ctx context.Context) (models.ResultsReport, error)
	CreateTimeReport(ctx context.Context) ([]models.TimeReport, error)
}
