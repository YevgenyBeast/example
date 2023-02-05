package ports

import (
	"context"
	"task/internal/domain/models"
)

// Task интерфейс для работы с задачами
type Task interface {
	CreateTask(ctx context.Context, task models.Task) (models.Task, error)
	UpdateTask(ctx context.Context, task models.Task, userLogin string) (models.Task, error)
	DeleteTask(ctx context.Context, taskID, userLogin string) error
	GetTaskByID(ctx context.Context, taskID string) (models.Task, error)
	GetAllTasks(ctx context.Context) ([]models.Task, error)
	ApprovalTask(ctx context.Context, taskID, userLogin string) error
	DeclineTask(ctx context.Context, taskID, userLogin string) error
}
