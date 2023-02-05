package http

import (
	"context"
	"net/http"
	"strconv"
	"task/internal/common"
	"task/internal/domain/models"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
)

// nolint
var (
	counterValidate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "team35", Subsystem: "task", Name: "validate_attempts", Help: "validate request counter",
		},
		[]string{"method", "endpoint", "code"})
)

func (s *Server) httpDebugMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viper.GetBool("profiler") {
			next.ServeHTTP(w, r)
		} else {
			responseHTTP(w, http.StatusOK, "pprof is off. need to make GET-request to: /switch?profiler=on")
		}
	})
}

func (s *Server) httpLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := common.InitLogger()
		ctx := r.Context()
		next.ServeHTTP(w, r.WithContext(ContextWithLogger(ctx, logger)))
	})
}

type loggerCtxKey struct{}

// ContextWithLogger добавить логгер в контекст
func ContextWithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// LoggerFromContext получить логгер из контекста
func LoggerFromContext(ctx context.Context) *logrus.Entry {
	return ctx.Value(loggerCtxKey{}).(*logrus.Entry)
}

func (s *Server) httpHostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		next.ServeHTTP(w, r.WithContext(ContextWithHost(ctx, r.Host)))
	})
}

type hostCtxKey struct{}

// ContextWithHost добавить адрес хоста в контекст
func ContextWithHost(ctx context.Context, host string) context.Context {
	return context.WithValue(ctx, hostCtxKey{}, host)
}

// HostFromContext получить адрес хоста из контекста
func HostFromContext(ctx context.Context) string {
	return ctx.Value(hostCtxKey{}).(string)
}

type userCtxKey struct{}

// ContextWithUser добавить данные пользователя в контекст
func ContextWithUser(ctx context.Context, user models.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, user)
}

// UserFromContext получить данные пользователя из контекста
func UserFromContext(ctx context.Context) models.User {
	return ctx.Value(userCtxKey{}).(models.User)
}

func (s *Server) validateMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "validateMiddleware")
		defer span.End()

		logger := LoggerFromContext(ctx)
		// Проверяем наличие access-куки
		accessCookie, err := r.Cookie(models.AccessCookie)
		if err != nil {
			logger.Errorf("invalid access-cookie: %s", err.Error())
			sentry.CaptureException(err)
			counterValidate.WithLabelValues(r.Method, r.RequestURI, strconv.Itoa(http.StatusForbidden)).Inc()
			responseHTTP(w, http.StatusForbidden, "invalid token")

			return
		}
		// Проверяем наличие refresh-куки
		refreshCookie, err := r.Cookie(models.RefreshCookie)
		if err != nil {
			logger.Errorf("invalid refresh-cookie: %s", err.Error())
			sentry.CaptureException(err)
			counterValidate.WithLabelValues(r.Method, r.RequestURI, strconv.Itoa(http.StatusForbidden)).Inc()
			responseHTTP(w, http.StatusForbidden, "invalid token")

			return
		}

		user, accessToken, refreshToken, err := s.auth.ValidateToken(ctx, accessCookie.Value, refreshCookie.Value)
		if err != nil {
			logger.Errorf("validate token: %s", err.Error())
			sentry.CaptureException(err)
			counterValidate.WithLabelValues(r.Method, r.RequestURI, strconv.Itoa(http.StatusForbidden)).Inc()
			responseHTTP(w, http.StatusForbidden, "invalid token")

			return
		}

		if accessToken != "" {
			setTokensToCookie(w, accessToken, refreshToken)
		}
		// Добавляем в контекст данные о пользователе
		r = r.WithContext(ContextWithUser(ctx, user))
		counterValidate.WithLabelValues(r.Method, r.RequestURI, strconv.Itoa(http.StatusOK)).Inc()
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func setTokensToCookie(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     models.AccessCookie,
		Value:    accessToken,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     models.RefreshCookie,
		Value:    refreshToken,
		HttpOnly: true,
	})
}
