package postgre

import (
	"context"
	"fmt"

	"analytics/internal/domain/models"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type PostgreDatabase struct {
	Pool *pgxpool.Pool
}

// New - конструктор адаптора подключения к БД
func New(ctx context.Context, postgreConn string) (*PostgreDatabase, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "PostgreDatabase")
	defer span.End()

	pool, err := pgxpool.Connect(ctx, postgreConn)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("postgre connect was failed: %s", err.Error())
	}

	return &PostgreDatabase{
		Pool: pool,
	}, nil
}
