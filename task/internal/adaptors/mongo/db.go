package mongo

import (
	"context"
	"fmt"
	"task/internal/domain/models"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Database struct {
	mClient *mongo.Client
	dbName  string
}

// New - конструктор адаптора подключения к БД
func New(ctx context.Context, mongoConn, dbName string) (*Database, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "MongoDatabase")
	defer span.End()

	cOpts := options.Client().ApplyURI(mongoConn)

	mClient, err := mongo.Connect(ctx, cOpts)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return nil, fmt.Errorf("mongo connect was failed: %s", err.Error())
	}

	ctxWithCancel, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = mClient.Ping(ctxWithCancel, nil)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return nil, fmt.Errorf("mongo ping was failed: %s", err.Error())
	}

	return &Database{
		mClient: mClient,
		dbName:  dbName,
	}, nil
}
