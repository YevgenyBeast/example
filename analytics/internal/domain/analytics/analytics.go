package analytics

import (
	"context"

	"analytics/internal/domain/models"
	"analytics/internal/ports"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type analyticsService struct {
	db ports.AnalyticsStorage
}

// NewAnalyticsService - конструктор сервиса аналитики
func NewAnalyticsService(db ports.AnalyticsStorage) *analyticsService {
	return &analyticsService{
		db: db,
	}
}

// CreateResultsReport формирует отчёт о результатах согласования задач
func (s *analyticsService) CreateResultsReport(ctx context.Context) (models.ResultsReport, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "CreateResultsReport")
	defer span.End()

	result, err := s.db.GetResults(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return models.ResultsReport{}, err
	}

	return result, nil
}

// CreateTimeReport формирует отчёт о времени согласования задач
func (s *analyticsService) CreateTimeReport(ctx context.Context) ([]models.TimeReport, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "CreateTimeReport")
	defer span.End()

	durationTask, err := s.db.GetDuration(ctx, "task")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	durationApprove, err := s.db.GetDuration(ctx, "approve")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	report := make([]models.TimeReport, 0, len(durationTask))
	for id, duration := range durationTask {
		report = append(report, models.TimeReport{
			TaskID:      id,
			ApproveTime: durationApprove[id].String(),
			TotalTime:   duration.String(),
		})
	}

	return report, nil
}
