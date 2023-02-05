package analytics

import (
	"context"
	"fmt"
	"task/internal/adaptors/client/analyticspb"
	"task/internal/domain/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	client analyticspb.AnalyticsClient
}

// NewAnalyticsService - конструктор
func NewAnalyticsService(client analyticspb.AnalyticsClient) *Service {
	return &Service{client}
}

// SendResult отправляет данные о результате согласования задачи в Analytics
func (s *Service) SendResult(ctx context.Context, data models.ResultData) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SendResult")
	defer span.End()

	rq := analyticspb.ResultRq{
		Taskid: data.TaskID,
		Result: data.Result,
	}

	_, err := s.client.SendResult(ctx, &rq)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return fmt.Errorf("invalid send result: %w", err)
	}

	return nil
}

// SendTimestamp отправляет данные о времени события в Analytics
// nolint
func (s *Service) SendTimestamp(ctx context.Context, data models.TimestampData) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SendTimestamp")
	defer span.End()

	rq := analyticspb.TimestampRq{
		Taskid:    data.TaskID,
		Approver:  data.Approver,
		Eventtype: string(data.EventType),
		Starttime: timestamppb.New(data.Start),
		Endtime:   timestamppb.New(data.End),
	}

	_, err := s.client.SendTimestamp(ctx, &rq)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return fmt.Errorf("invalid send timestamp: %w", err)
	}

	return nil
}
