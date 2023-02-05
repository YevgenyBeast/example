package ports

import (
	"context"
	"task/internal/domain/models"
)

// Analytics интерфейс для отправки данных в Analytics
type Analytics interface {
	SendResult(ctx context.Context, data models.ResultData) error
	SendTimestamp(ctx context.Context, data models.TimestampData) error
}
