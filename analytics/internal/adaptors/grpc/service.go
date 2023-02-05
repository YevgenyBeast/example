package grpc

import (
	"context"

	"analytics/internal/domain/models"
	"analytics/internal/ports"
	"analytics/pkg/analyticspb"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	analytics ports.AnalyticsStorage
	analyticspb.UnimplementedAnalyticsServer
}

// New возвращает имплементацию gRPC сервиса
func New(analytics ports.AnalyticsStorage) analyticspb.AnalyticsServer {
	return &Service{
		analytics: analytics,
	}
}

// SendResult метод для отправки данных о результате согласования задачи в Analytics
func (srv *Service) SendResult(ctx context.Context, rq *analyticspb.ResultRq) (*emptypb.Empty, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SendResult gRPC")
	defer span.End()

	err := srv.analytics.SetResult(ctx, models.ResultData{
		TaskID: rq.Taskid,
		Result: rq.Result,
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, err
}

// SendTimestamp метод для отправки данных о времени события в Analytics
func (srv *Service) SendTimestamp(ctx context.Context, rq *analyticspb.TimestampRq) (*emptypb.Empty, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SendTimestamp gRPC")
	defer span.End()

	err := srv.analytics.SetTimestamp(ctx, models.TimestampData{
		TaskID:    rq.Taskid,
		Approver:  rq.Approver,
		EventType: models.Event(rq.Eventtype),
		Start:     rq.Starttime.AsTime(),
		End:       rq.Endtime.AsTime(),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, err
}
