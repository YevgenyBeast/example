package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"analytics/internal/ports"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag/example/basic/docs"
)

// @title Analytics API
// @version 1.0
// @description API Server for Analytics Application

// @contact.name Alyoshkin Yevgeny
// @contact.email alyevgenyal@mail.ru

// Server содержит httpServer
type Server struct {
	httpServer *http.Server
	analytics  ports.Analytics
	auth       ports.Auth
}

// NewServer конструктор http-сервера
func NewServer(port string, analytics ports.Analytics, auth ports.Auth) *Server {
	var s Server
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: s.routes(),
	}
	s.analytics = analytics
	s.auth = auth
	return &s
}

// NewServer конструктор сервера для сбора метрик
func NewServerMetrics(port string) *Server {
	var s Server
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: nil,
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

	r.Mount("/", s.analyticsHandlers())

	r.Get("/switch", s.switcher)
	r.With(s.httpDebugMiddleware).Mount("/debug", middleware.Profiler())

	docs.SwaggerInfo.Host = viper.GetString("swagger-host")
	docs.SwaggerInfo.BasePath = viper.GetString("swagger-base-path")
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("%s/swagger/doc.json", viper.GetString("swagger-base-path")))))

	http.Handle("/metrics", promhttp.Handler())

	return r
}

func responseHttp(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	je := json.NewEncoder(w)
	_ = je.Encode(data)
}

// switcher godoc
// @Summary switcher
// @Tags analytics
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
		responseHttp(w, http.StatusOK, "pprof was run")
		return
	}
	if flag == "off" {
		viper.Set("profiler", false)
		logger.Info("pprof was stop")
		responseHttp(w, http.StatusOK, "pprof was stop")
		return
	}
}
