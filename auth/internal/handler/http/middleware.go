package http

import (
	"context"
	"net/http"
	"strconv"

	"auth/internal/common"
	"auth/internal/model"

	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type userCtxKey struct{}

func (h Handler) httpUserMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer(model.TracerName).Start(r.Context(), "httpUserMiddleware")
		defer span.End()

		logger := LoggerFromContext(ctx)
		// Проверяем наличие access-куки
		accessCookie, err := r.Cookie(model.AccessCookie)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			logger.Errorf("parse request no access-cookie: %s", err.Error())
			sentry.CaptureException(err)
			counterValidate.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
			responseHttp(w, http.StatusForbidden, "parse request: no access-cookie")
			return
		}
		// Проверяем наличие refresh-куки
		refreshCookie, err := r.Cookie(model.RefreshCookie)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			logger.Errorf("parse request no refresh-cookie: %s", err.Error())
			sentry.CaptureException(err)
			counterValidate.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
			responseHttp(w, http.StatusForbidden, "parse request: no refresh-cookie")
			return
		}

		user, accessToken, refreshToken, err := h.dc.AuthService.ValidateTokens(ctx, accessCookie.Value, refreshCookie.Value)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			logger.Errorf("parse token request: %s", err.Error())
			sentry.CaptureException(err)
			counterValidate.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
			responseHttp(w, http.StatusForbidden, "parse token request: invalid")
			return
		}
		if accessToken != "" {
			setTokensToCookie(w, accessToken, refreshToken)
		}

		// Добавляем в контекст данные о пользователе
		r = r.WithContext(context.WithValue(ctx, userCtxKey{}, *user))
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (h Handler) httpDebugMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viper.GetBool("profiler") {
			next.ServeHTTP(w, r)
		} else {
			responseHttp(w, http.StatusOK, "pprof is off. need make GET-request to endpoint: /switch?profiler=on")
		}
	})
}

func (h Handler) httpLoggerMiddleware(next http.Handler) http.Handler {
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
