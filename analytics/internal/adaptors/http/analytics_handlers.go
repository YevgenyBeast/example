package http

import (
	"net/http"

	"analytics/internal/domain/models"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

func (s *Server) analyticsHandlers() http.Handler {
	r := chi.NewRouter()

	r.Route("/analytics", func(r chi.Router) {
		r.Use(s.validateMiddleware)

		r.Get("/results", s.getResultsReport)
		r.Get("/time", s.getTimeReport)
	})

	return r
}

// getResultsReport godoc
// @Summary getResultsReport
// @Tags analytics
// @Description generates a report on the result of completed tasks
// @Produce json
// @Success 200 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /analytics/results [get]
func (s *Server) getResultsReport(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "getResultsReport")
	defer span.End()

	logger := LoggerFromContext(ctx)

	result, err := s.analytics.CreateResultsReport(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Errorf("create results report error: %s", err.Error())
		sentry.CaptureException(err)
		responseHttp(w, http.StatusInternalServerError, "create results report was failed")
		return
	}
	responseHttp(w, http.StatusOK, result)
	logger.Infof("report about tasks'results was created")
}

// getTimeReport godoc
// @Summary getTimeReport
// @Tags analytics
// @Description generates a report on the total time and approval time
// @Produce json
// @Success 200 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /analytics/time [get]
func (s *Server) getTimeReport(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "getTimeReport")
	defer span.End()

	logger := LoggerFromContext(ctx)

	result, err := s.analytics.CreateTimeReport(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Errorf("create time report error: %s", err.Error())
		sentry.CaptureException(err)
		responseHttp(w, http.StatusInternalServerError, "create time report was failed")
		return
	}
	responseHttp(w, http.StatusOK, result)
	logger.Infof("report about approval and total time was created")
}
