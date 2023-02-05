package action

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"auth/internal/model"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const (
	salt       = "keL73!ox*2ahm"
	signingKey = "vq<MdCk#y50lS"
	accessTTL  = 1 * time.Minute
	refreshTTL = 1 * time.Hour
)

type authService struct {
	repo AuthRepository
}

// AuthRepository для управления авторизацией
type AuthRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, login, hashpswrd string) (*model.User, error)
}

type tokenClaims struct {
	jwt.StandardClaims
	model.User
}

// NewAuthService - конструктор сервиса авторизации
func NewAuthService(repo AuthRepository) *authService {
	return &authService{repo: repo}
}

// Login проверяет логин и пароль пользователя и генерирует access и refresh токены
func (s *authService) Login(ctx context.Context, login, password string) (accessToken, refreshToken string, err error) {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "Login")
	defer span.End()

	user, err := s.repo.GetUser(ctx, login, generatePasswordHash(ctx, password))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}

	return s.generateTokens(ctx, user)
}

func (s *authService) ValidateTokens(ctx context.Context, accessToken, refreshToken string) (*model.User, string, string, error) {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "ValidateTokens")
	defer span.End()

	//Проверяем access-token
	user, err := s.parseToken(ctx, accessToken)
	if err != nil {
		// Если в access-token есть ошибки или он просрочен, то проверяем refresh-token
		user, err = s.parseToken(ctx, refreshToken)
		if err != nil {
			// Если в refresh-token есть ошибки или он просрочен, то возвращаем ошибку
			span.SetStatus(codes.Error, err.Error())
			return nil, "", "", errors.WithStack(err)
		}
		accessToken, refreshToken, err = s.generateTokens(ctx, user)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, "", "", errors.WithStack(err)
		}
		return user, accessToken, refreshToken, nil
	}
	return user, "", "", nil
}

// GenerateTokens генерирует access и refresh токены
func (s *authService) generateTokens(ctx context.Context, user *model.User) (accessToken, refreshToken string, err error) {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "generateTokens")
	defer span.End()

	accessToken, err = generateToken(ctx, user, accessTTL)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", "", errors.WithStack(err)
	}

	refreshToken, err = generateToken(ctx, user, refreshTTL)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", "", errors.WithStack(err)
	}

	return accessToken, refreshToken, nil
}

func generateToken(ctx context.Context, user *model.User, ttl time.Duration) (string, error) {
	_, span := otel.Tracer(model.TracerName).Start(ctx, "generateToken")
	defer span.End()

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(ttl).Unix(),
		},
		model.User{
			Username: user.Username,
			Email:    user.Email,
		},
	})

	signedToken, err := unsignedToken.SignedString([]byte(signingKey))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return signedToken, nil
}

// ParseToken проверяет валидность токена
func (s *authService) parseToken(ctx context.Context, token string) (*model.User, error) {
	_, span := otel.Tracer(model.TracerName).Start(ctx, "parseToken")
	defer span.End()

	parseToken, err := jwt.ParseWithClaims(token, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			span.SetStatus(codes.Error, "invalid signing method")
			return nil, errors.New("invalid signing method")
		}
		return []byte(signingKey), nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}
	if !parseToken.Valid {
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.New("token is expired")
	}

	claims, ok := parseToken.Claims.(*tokenClaims)
	if !ok {
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.New("token claims have invalid type")
	}

	return &claims.User, nil
}

func generatePasswordHash(ctx context.Context, password string) string {
	_, span := otel.Tracer(model.TracerName).Start(ctx, "generatePasswordHash")
	defer span.End()

	hash := md5.New()
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum([]byte(salt)))
}

func (s *authService) CreateUser(ctx context.Context, user *model.User) error {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "CreateUser")
	defer span.End()

	user.Password = generatePasswordHash(ctx, user.Password)
	return s.repo.CreateUser(ctx, user)
}
