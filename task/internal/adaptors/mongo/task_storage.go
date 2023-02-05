package mongo

import (
	"context"
	"fmt"
	"task/internal/domain/errors"
	"task/internal/domain/models"
	"task/internal/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel"
)

const (
	collName = "tasks"
)

var _ ports.TaskStorage = (*Database)(nil)

// InsertTask сохраняет новую задачу в БД
func (db *Database) InsertTask(ctx context.Context, task models.Task) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "InsertTask")
	defer span.End()

	mCollection := db.mClient.Database(db.dbName).Collection(collName)

	_, err := mCollection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("insert task was failed: %w", err)
	}

	return nil
}

// UpdateTask обновляет данные задачи в БД
func (db *Database) UpdateTask(ctx context.Context, task models.Task) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "UpdateTask")
	defer span.End()

	mCollection := db.mClient.Database(db.dbName).Collection(collName)

	filter := bson.M{
		"_id": task.ID,
	}
	update := bson.M{
		"$set": task,
	}

	_, err := mCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update task was failed: %w", err)
	}

	return nil
}

// DeleteTask обновляет данные задачи в БД
func (db *Database) DeleteTask(ctx context.Context, id string) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "DeleteTask")
	defer span.End()

	mCollection := db.mClient.Database(db.dbName).Collection(collName)

	filter := bson.M{
		"_id": id,
	}

	_, err := mCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("delete task was failed: %w", err)
	}

	return nil
}

// GetTask ищет задачу в БД по ID
func (db *Database) GetTask(ctx context.Context, id string) (models.Task, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "GetTask")
	defer span.End()

	mCollection := db.mClient.Database(db.dbName).Collection(collName)

	filter := bson.M{
		"_id": id,
	}

	var task models.Task

	err := mCollection.FindOne(ctx, filter).Decode(&task)
	if err != nil {
		return models.Task{}, errors.ErrTaskNotFound
	}

	return task, nil
}

// GetTasks ищет задачу в БД по ID
func (db *Database) GetTasks(ctx context.Context) ([]models.Task, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "GetTasks")
	defer span.End()

	mCollection := db.mClient.Database(db.dbName).Collection(collName)
	filter := bson.M{}

	var tasks []models.Task

	res, err := mCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("get tasks was failed: %w", err)
	}

	err = res.All(ctx, &tasks)
	if err != nil {
		return nil, fmt.Errorf("get tasks was failed: %w", err)
	}

	return tasks, nil
}
