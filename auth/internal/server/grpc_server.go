package server

import (
	"auth/internal/action"
	grpc_service "auth/internal/handler/grpc"
	"auth/pkg/authpb"
	"errors"
	"net"

	"google.golang.org/grpc"
)

// ServerGRPC содержит grpc.Server
type ServerGRPC struct {
	grpcServer *grpc.Server
}

func NewGRPCServer(di action.Container) *ServerGRPC {
	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServer(grpcServer, grpc_service.New(di))

	return &ServerGRPC{
		grpcServer: grpcServer,
	}
}

// Run запускает сервер
func (s *ServerGRPC) Run(port string) error {
	port = ":" + port
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	if err := s.grpcServer.Serve(lis); !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}

	return nil
}

// Shutdown останавливает сервер
func (s *ServerGRPC) Stop() {
	s.grpcServer.GracefulStop()
}
