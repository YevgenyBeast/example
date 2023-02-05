package server

import (
	"context"
	"net/http"
)

// Server содержит httpServer
type Server struct {
	httpServer *http.Server
}

// Run запускает сервер
func (s *Server) Run(handler http.Handler, port string) error {
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
