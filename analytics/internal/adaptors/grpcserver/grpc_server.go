package grpcserver

import (
	"errors"
	"net"

	grpc_service "analytics/internal/adaptors/grpc"
	"analytics/internal/ports"
	"analytics/pkg/analyticspb"

	"google.golang.org/grpc"
)

// ServerGRPC содержит grpc.Server
type ServerGRPC struct {
	grpcServer *grpc.Server
}

// NewGRPCServer конструктор ServerGRPC
func NewGRPCServer(analytics ports.AnalyticsStorage) *ServerGRPC {
	grpcServer := grpc.NewServer()
	analyticspb.RegisterAnalyticsServer(grpcServer, grpc_service.New(analytics))

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
