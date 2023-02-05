package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"task/docs"
	"task/internal/ports"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Task API
// @version 1.0
// @description API Server for Task Application

// @contact.name Alyoshkin Yevgeny
// @contact.email alyevgenyal@mail.ru

// Server содержит httpServer
type Server struct {
	httpServer *http.Server
	task       ports.Task
	auth       ports.Auth
}

// NewServer конструктор http-сервера
func NewServer(port string, task ports.Task, auth ports.Auth) *Server {
	var s Server
	s.httpServer = &http.Server{
		Addr:              ":" + port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.task = task
	s.auth = auth

	return &s
}

// NewServer конструктор сервера для сбора метрик
func NewServerMetrics(port string) *Server {
	var s Server
	s.httpServer = &http.Server{
		Addr:              ":" + port,
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &s
}

// Run запускает сервер
func (s *Server) Run() error {
	if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(s.httpLoggerMiddleware)
	r.Use(s.httpHostMiddleware)

	r.Mount("/", s.taskHandlers())

	r.Get("/switch", s.switcher)
	r.With(s.httpDebugMiddleware).Mount("/debug", middleware.Profiler())

	docs.SwaggerInfo.Host = viper.GetString("swagger-host")
	docs.SwaggerInfo.BasePath = viper.GetString("swagger-base-path")
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("%s/swagger/doc.json", viper.GetString("swagger-base-path")))))

	http.Handle("/metrics", promhttp.Handler())

	return r
}

func responseHTTP(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	je := json.NewEncoder(w)
	_ = je.Encode(data)
}

// switcher godoc
// @Summary switcher
// @Tags task
// @Description switch debug mode
// @Accept json
// @Produce json
// @Param profiler query string true "switch debug" example(on)
// @Success 200 {string} string
// @Router /switch [get]
func (s *Server) switcher(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := LoggerFromContext(ctx)
	flag := r.URL.Query().Get("profiler")

	if flag == "on" {
		viper.Set("profiler", true)
		logger.Info("pprof was run")
		responseHTTP(w, http.StatusOK, "pprof was run")

		return
	}

	if flag == "off" {
		viper.Set("profiler", false)
		logger.Info("pprof was stop")
		responseHTTP(w, http.StatusOK, "pprof was stop")

		return
	}
}
