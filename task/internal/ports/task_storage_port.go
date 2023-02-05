package ports

import (
	"context"
	"task/internal/domain/models"
)

// TaskSrorage интерфейс для работы с БД
type TaskStorage interface {
	InsertTask(ctx context.Context, task models.Task) error
	UpdateTask(ctx context.Context, task models.Task) error
	DeleteTask(ctx context.Context, id string) error
	GetTask(ctx context.Context, id string) (models.Task, error)
	GetTasks(ctx context.Context) ([]models.Task, error)
}
