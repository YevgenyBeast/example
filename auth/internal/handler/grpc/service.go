package grpc

import (
	"auth/internal/action"
	"auth/internal/model"
	"auth/pkg/authpb"
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Service struct {
	dc action.Container
	authpb.UnimplementedAuthServer
}

// New возвращает имплементацию gRPC сервиса
func New(dc action.Container) authpb.AuthServer {
	return &Service{
		dc: dc,
	}
}

func (srv *Service) Validate(ctx context.Context, rq *authpb.ValidateRq) (
	*authpb.ValidateRs, error) {
	ctx, span := otel.Tracer(model.TracerName).Start(ctx, "Validate gRPC")
	defer span.End()

	user, accessToken, refreshToken, err := srv.dc.ValidateTokens(ctx, rq.Access, rq.Refresh)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("validate by gRPC was failed: %w", err)
	}
	return &authpb.ValidateRs{
		User: &authpb.User{
			UserLogin: user.Username,
			UserEmail: user.Email,
		},
		Access:  accessToken,
		Refresh: refreshToken,
	}, nil
}
