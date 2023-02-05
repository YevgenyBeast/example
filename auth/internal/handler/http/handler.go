package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"auth/docs"
	"auth/internal/action"
	"auth/internal/model"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// @title Auth API
// @version 1.0
// @description API Server for Auth Application

// @contact.name Alyoshkin Yevgeny
// @contact.email alyevgenyal@mail.ru

// @securityDefinitions.basic BasicAuth

var (
	counterLogin = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "team35", Subsystem: "auth", Name: "login_attempts", Help: "/login endpoint request counter",
		},
		[]string{"code"})
	counterValidate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "team35", Subsystem: "auth", Name: "validate_attempts", Help: "/validate endpoint request counter",
		},
		[]string{"code"})
)

// Handler содержит контейнер с доступными сервисами
type Handler struct {
	dc action.Container
}

// NewHandler создание нового Handler
func NewHandler(dc action.Container) http.Handler {
	h := Handler{dc: dc}
	return h.newRouter()
}

func (h Handler) newRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(h.httpLoggerMiddleware)

	r.Route("/", func(r chi.Router) {
		r.Post("/create", h.create)
		r.Post("/login", h.login)
		r.Post("/logout", h.logout)
		r.With(h.httpUserMiddleware).Get("/i", h.getInformation)
		r.Get("/switch", h.switcher)
	})

	docs.SwaggerInfo.Host = viper.GetString("swagger-host")
	docs.SwaggerInfo.BasePath = viper.GetString("swagger-base-path")
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("%s/swagger/doc.json", viper.GetString("swagger-base-path")))))

	r.With(h.httpDebugMiddleware).Mount("/debug", middleware.Profiler())

	http.Handle("/metrics", promhttp.Handler())

	return r
}

// switcher godoc
// @Summary switcher
// @Tags auth
// @Description switch debug mode
// @Accept json
// @Produce json
// @Param profiler query string true "switch debug" example(on)
// @Success 200 {string} string
// @Router /switch [get]
func (h Handler) switcher(w http.ResponseWriter, r *http.Request) {
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

// create godoc
// @Summary create
// @Tags auth
// @Description create user in BD
// @Accept json
// @Produce json
// @Param userReq body model.User true "User data"
// @Success 200 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /create [post]
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(model.TracerName).Start(r.Context(), "create")
	defer span.End()

	logger := LoggerFromContext(ctx)

	var userReq model.User
	err := json.NewDecoder(r.Body).Decode(&userReq)
	defer r.Body.Close()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Errorf("user create error: %s", err.Error())
		sentry.CaptureException(err)
		responseHttp(w, http.StatusInternalServerError, "user create error")
		return
	}
	if userReq.Username == "" || userReq.Password == "" {
		span.SetStatus(codes.Error, err.Error())
		logger.Errorf("user create error: empty user-data")
		sentry.CaptureException(fmt.Errorf("user create error: empty user-data"))
		responseHttp(w, http.StatusBadRequest, "user create error")
		return
	}

	err = h.dc.AuthService.CreateUser(ctx, &userReq)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Errorf("user create error: %s", err.Error())
		sentry.CaptureException(err)
		responseHttp(w, http.StatusInternalServerError, "user create error")
		return
	}

	responseHttp(w, http.StatusOK, "user was created")
	logger.Infof("user: %s was created", userReq.Username)
}

// login godoc
// @Summary login
// @Tags auth
// @Description login user
// @Accept json
// @Produce json
// @Param Authorization header string true "BasicAuth" example(Basic VGVzdFVzZXI6RGVyUGFyb2w=)
// @Success 200 {object} model.AuthResponse
// @Failure 403 {string} string
// @Router /login [post]
func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(model.TracerName).Start(r.Context(), "login")
	defer span.End()

	logger := LoggerFromContext(ctx)

	login, password, ok := r.BasicAuth()
	if !ok {
		span.SetStatus(codes.Error, "login internal error")
		logger.Errorf("login internal error")
		sentry.CaptureException(fmt.Errorf("login internal error"))
		counterLogin.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
		responseHttp(w, http.StatusForbidden, "login internal error")
		return
	}
	authReq := model.AuthRequest{
		Login:    login,
		Password: password,
	}

	accessToken, refreshToken, err := h.dc.AuthService.Login(ctx, authReq.Login, authReq.Password)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Errorf("login failed: %s", err.Error())
		sentry.CaptureException(err)
		counterLogin.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
		responseHttp(w, http.StatusForbidden, "login failed")
		return
	}
	setTokensToCookie(w, accessToken, refreshToken)

	authRes := model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ifRedirectIsNeed(w, r)
	counterLogin.WithLabelValues(strconv.Itoa(http.StatusOK)).Inc()
	responseHttp(w, http.StatusOK, authRes)
	logger.Infof("user: %s was login", authReq.Login)
}

// logout godoc
// @Summary logout
// @Tags auth
// @Description logout user
// @Produce json
// @Success 200 {string} string
// @Router /logout [post]
func (h Handler) logout(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(model.TracerName).Start(r.Context(), "logout")
	defer span.End()

	logger := LoggerFromContext(ctx)

	http.SetCookie(w, &http.Cookie{
		Name:     model.AccessCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     model.RefreshCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	ifRedirectIsNeed(w, r)
	responseHttp(w, http.StatusOK, "logout")
	logger.Infof("logout")
}

// getInformation godoc
// @Summary getInformation
// @Tags auth
// @Description get information about logged user
// @Produce json
// @Success 200 {object} model.User
// @Failure 403 {string} string
// @Router /i [get]
func (h Handler) getInformation(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(model.TracerName).Start(r.Context(), "getInformation")
	defer span.End()

	logger := LoggerFromContext(ctx)
	user, ok := ctx.Value(userCtxKey{}).(model.User)
	if !ok {
		span.SetStatus(codes.Error, "get information: no user information in request")
		logger.Error("get information: no user information in request")
		sentry.CaptureException(fmt.Errorf("get information: no user information in request"))
		counterValidate.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
		responseHttp(w, http.StatusForbidden, "get information: no user information in request")
		return
	}
	logger.Infof("user: %s requested information", user.Username)
	counterValidate.WithLabelValues(strconv.Itoa(http.StatusOK)).Inc()
	responseHttp(w, http.StatusOK, user)
}

func responseHttp(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	je := json.NewEncoder(w)
	_ = je.Encode(data)
}

func ifRedirectIsNeed(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer(model.TracerName).Start(r.Context(), "ifRedirectIsNeed")
	defer span.End()

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI != "" {
		http.Redirect(w, r, redirectURI, http.StatusMovedPermanently)
	}
}

func setTokensToCookie(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     model.AccessCookie,
		Value:    accessToken,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     model.RefreshCookie,
		Value:    refreshToken,
		HttpOnly: true,
	})
}
