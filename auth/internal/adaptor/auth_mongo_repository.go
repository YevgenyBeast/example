package adaptor

import (
	"auth/internal/model"
	"context"

	uuid "github.com/satori/go.uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const (
	collName = "users"
)

type authMongoRepository struct {
	mClient *mongo.Client
	dbName  string
}

// NewAuthMemoryRepository - конструктор репозитория авторизации
func NewAuthMongoRepository(ctx context.Context, mongoConn, dbName string) (*authMongoRepository, error) {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "NewAuthMongoRepository")
	defer span.End()

	cOpts := options.Client().ApplyURI(mongoConn)
	mClient, err := mongo.Connect(ctx, cOpts)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &authMongoRepository{
		mClient: mClient,
		dbName:  dbName,
	}, nil
}

// CreateUser - добавляет пользователя в БД
func (r *authMongoRepository) CreateUser(ctx context.Context, user *model.User) error {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "CreateUser")
	defer span.End()

	mCollection := r.mClient.Database(r.dbName).Collection(collName)

	user.ID = uuid.NewV4().String()
	_, err := mCollection.InsertOne(ctx, *user)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// GetUser - получение данных о пользователе
func (r *authMongoRepository) GetUser(ctx context.Context, login, pswrdhash string) (*model.User, error) {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "GetUser")
	defer span.End()

	mCollection := r.mClient.Database(r.dbName).Collection(collName)
	filter := bson.M{
		"username":     login,
		"passwordhash": pswrdhash,
	}

	var user model.User
	err := mCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, model.ErrAuthentication
	}

	return &user, nil
}
