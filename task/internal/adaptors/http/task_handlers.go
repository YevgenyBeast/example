package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	taskErrors "task/internal/domain/errors"
	"task/internal/domain/models"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
)

func (s *Server) taskHandlers() http.Handler {
	r := chi.NewRouter()

	r.Route("/tasks", func(r chi.Router) {
		r.Use(s.validateMiddleware)

		r.Get("/", s.getAllTasks)
		r.Post("/run", s.create)
		r.Put("/{id}", s.update)
		r.Get("/{id}", s.getTaskByID)
		r.Delete("/{id}", s.delete)
		r.Post("/{id}/approve/{login}", s.approve)
		r.Post("/{id}/decline/{login}", s.decline)
	})

	return r
}

// create godoc
// @Summary create
// @Tags task
// @Description create task and run it
// @Accept json
// @Produce json
// @Param task body models.Task true "Task"
// @Success 200 {object} models.TaskRes
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /tasks/run [post]
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "create")
	defer span.End()

	logger := LoggerFromContext(ctx)
	user := UserFromContext(ctx).UserLogin

	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)

	defer r.Body.Close()

	if err != nil {
		logger.Errorf("task create error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, http.StatusBadRequest, "task create was failed")

		return
	}

	if user != task.InitiatorLogin {
		logger.Errorf("task create error: invalid initiator's token")
		sentry.CaptureException(fmt.Errorf("task create error: invalid initiator's token"))
		responseHTTP(w, http.StatusForbidden, "task create was failed")

		return
	}

	task, err = s.task.CreateTask(ctx, task)
	if err != nil {
		logger.Errorf("task create error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, http.StatusInternalServerError, "task create was failed")

		return
	}

	taskRes := makeTaskRes(task)

	responseHTTP(w, http.StatusOK, taskRes)
	logger.Infof("task: %s was created", task.ID)
}

func makeTaskRes(task models.Task) models.TaskRes {
	var taskRes models.TaskRes
	taskRes.ID = task.ID
	taskRes.InitiatorLogin = task.InitiatorLogin

	for _, val := range task.ApprovalLogins {
		taskRes.Approval = append(taskRes.Approval, models.Approval{Login: val})
	}

	return taskRes
}

// update godoc
// @Summary update
// @Tags task
// @Description update task and run it again
// @Accept json
// @Produce json
// @Param id path string true "TaskID"
// @Success 200 {object} models.TaskRes
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /tasks/{id} [put]
func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "update")
	defer span.End()

	logger := LoggerFromContext(ctx)
	user := UserFromContext(ctx).UserLogin

	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)

	defer r.Body.Close()

	if err != nil {
		logger.Errorf("task update error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, http.StatusBadRequest, "task update was failed")

		return
	}

	task.ID = path.Base(r.URL.Path)

	task, err = s.task.UpdateTask(ctx, task, user)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		if errors.Is(err, taskErrors.ErrInitiatorInvalid) || errors.Is(err, taskErrors.ErrTaskNotFound) {
			httpStatus = http.StatusBadRequest
		}

		logger.Errorf("task update error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, httpStatus, "task update was failed")

		return
	}

	taskRes := makeTaskRes(task)

	responseHTTP(w, http.StatusOK, taskRes)
	logger.Infof("task: %s was updated", task.ID)
}

// getTaskByID godoc
// @Summary getTaskByID
// @Tags task
// @Description get task by id from DB
// @Accept json
// @Produce json
// @Param id path string true "TaskID"
// @Success 200 {object} models.TaskRes
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Router /tasks/{id} [get]
func (s *Server) getTaskByID(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "getTaskByID")
	defer span.End()

	logger := LoggerFromContext(ctx)

	taskID := path.Base(r.URL.Path)

	task, err := s.task.GetTaskByID(ctx, taskID)
	if err != nil {
		logger.Errorf("get task error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, http.StatusBadRequest, "get task was failed")

		return
	}

	responseHTTP(w, http.StatusOK, task)
	logger.Infof("get task: %s", taskID)
}

// getAllTasks godoc
// @Summary getAllTasks
// @Tags task
// @Description get all tasks from DB
// @Produce json
// @Success 200 {array} models.TaskRes
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /tasks/ [get]
func (s *Server) getAllTasks(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "getAllTasks")
	defer span.End()

	logger := LoggerFromContext(ctx)

	tasks, err := s.task.GetAllTasks(ctx)
	if err != nil {
		logger.Errorf("get all tasks error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, http.StatusInternalServerError, "get all tasks was failed")

		return
	}

	responseHTTP(w, http.StatusOK, tasks)
	logger.Infof("get all task")
}

// delete godoc
// @Summary delete
// @Tags task
// @Description delete task by id from DB
// @Accept json
// @Produce json
// @Param id path string true "TaskID"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /tasks/{id} [delete]
func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "delete")
	defer span.End()

	logger := LoggerFromContext(ctx)
	user := UserFromContext(ctx).UserLogin

	taskID := path.Base(r.URL.Path)

	err := s.task.DeleteTask(ctx, taskID, user)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		if errors.Is(err, taskErrors.ErrInitiatorInvalid) || errors.Is(err, taskErrors.ErrTaskNotFound) {
			httpStatus = http.StatusBadRequest
		}

		logger.Errorf("task delete error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, httpStatus, "task delete was failed")

		return
	}

	responseHTTP(w, http.StatusOK, "delete task")
	logger.Infof("task: %s was deleted", taskID)
}

// approve godoc
// @Summary approve
// @Tags task
// @Description approve task
// @Accept json
// @Produce json
// @Param id path string true "TaskID"
// @Param login path string true "Approval Login"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /tasks/{id}/approve/{login} [post]
// nolint
func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "approve")
	defer span.End()

	logger := LoggerFromContext(ctx)
	user := UserFromContext(ctx).UserLogin

	approver := path.Base(r.URL.Path)
	taskID := strings.Split(r.URL.Path, "/")[2]

	if user != approver {
		logger.Errorf("task approve error: invalid approver's token")
		sentry.CaptureException(fmt.Errorf("task approve error: invalid approver's token"))
		responseHTTP(w, http.StatusForbidden, "task approve was failed")
		return
	}

	err := s.task.ApprovalTask(ctx, taskID, approver)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		if errors.Is(err, taskErrors.ErrApprovalInvalid) || errors.Is(err, taskErrors.ErrTaskNotFound) {
			httpStatus = http.StatusBadRequest
		}
		logger.Errorf("task approve error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, httpStatus, "task approve was failed")
		return
	}
	responseHTTP(w, http.StatusOK, "approve task")
	logger.Infof("task: %s was approved by: %s", taskID, approver)
}

// decline godoc
// @Summary decline
// @Tags task
// @Description decline task
// @Accept json
// @Produce json
// @Param id path string true "TaskID"
// @Param login path string true "Approval Login"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /tasks/{id}/decline/{login} [post]
// nolint
func (s *Server) decline(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer(models.TracerName).Start(r.Context(), "decline")
	defer span.End()

	logger := LoggerFromContext(ctx)
	user := UserFromContext(ctx).UserLogin

	approver := path.Base(r.URL.Path)
	taskID := strings.Split(r.URL.Path, "/")[2]

	if user != approver {
		logger.Errorf("task decline error: invalid approver's token")
		sentry.CaptureException(fmt.Errorf("task decline error: invalid approver's token"))
		responseHTTP(w, http.StatusForbidden, "task decline was failed")
		return
	}

	err := s.task.DeclineTask(ctx, taskID, approver)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		if errors.Is(err, taskErrors.ErrApprovalInvalid) || errors.Is(err, taskErrors.ErrTaskNotFound) {
			httpStatus = http.StatusBadRequest
		}
		logger.Errorf("task decline error: %s", err.Error())
		sentry.CaptureException(err)
		responseHTTP(w, httpStatus, "task decline was failed")
		return
	}
	responseHTTP(w, http.StatusOK, "decline task")
	logger.Infof("task: %s was declined by: %s", taskID, approver)
}
