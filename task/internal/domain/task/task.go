package task

import (
	"context"
	"fmt"
	"task/internal/adaptors/http"
	taskErrors "task/internal/domain/errors"
	"task/internal/domain/models"
	"task/internal/ports"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Service struct {
	db        ports.TaskStorage
	mail      ports.Mail
	analytica ports.Analytics
}

// NewTaskService - конструктор сервиса задач
func NewTaskService(db ports.TaskStorage, mail ports.Mail, analytica ports.Analytics) *Service {
	return &Service{
		db:        db,
		mail:      mail,
		analytica: analytica,
	}
}

// CreateTask создать задачу
func (s *Service) CreateTask(ctx context.Context, task models.Task) (models.Task, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "CreateTask")
	defer span.End()

	task.ID = uuid.NewV4().String()
	task.CurrentApprovalNumber = 0
	task.Status = models.InProgressTaskStatus

	// Сохраняем задачу в БД
	err := s.db.InsertTask(ctx, task)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return models.Task{}, err
	}

	// Отправляем информацию о создании задачи в Analytics
	err = s.analytica.SendTimestamp(ctx, models.TimestampData{
		TaskID:    task.ID,
		EventType: models.TaskTypeEvent,
		Start:     time.Now().UTC(),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return models.Task{}, err
	}

	// Генерируем ссылки и отправляем письмо 1-ому согласующему
	links := generateLink(ctx, task)
	mail := models.MailToApproval{
		Destination: task.ApprovalLogins[task.CurrentApprovalNumber],
		ApproveLink: links.ApprovalLink,
		DeclineLink: links.DeclineLink,
	}
	// Отправляем письмо 1-ому согласующему
	err = s.mail.SendApprovalMail(ctx, mail)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return models.Task{}, err
	}

	// Отправляем информацию об отправке письма 1-ому согласующему в Analytics
	err = s.analytica.SendTimestamp(ctx, models.TimestampData{
		TaskID:    task.ID,
		Approver:  task.ApprovalLogins[task.CurrentApprovalNumber],
		EventType: models.ApproveTypeEvent,
		Start:     time.Now().UTC(),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return models.Task{}, err
	}

	return task, nil
}

// UpdateTask обновить задачу
func (s *Service) UpdateTask(ctx context.Context, task models.Task, userLogin string) (models.Task, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "UpdateTask")
	defer span.End()

	// Находим task по ID
	updTask, err := s.db.GetTask(ctx, task.ID)
	if err != nil {
		return models.Task{}, err
	}

	// Проверяем автора задачи
	if updTask.InitiatorLogin != userLogin {
		return models.Task{}, taskErrors.ErrInitiatorInvalid
	}

	// Готовим шаблон письма об обновлении задачи
	mailResult := models.ResultMail{
		Destinations: updTask.ApprovalLogins,
		TaskID:       updTask.ID,
		Result:       "task was updated",
	}

	// Сохраняем изменения в задаче
	updTask.ApprovalLogins = task.ApprovalLogins
	updTask.CurrentApprovalNumber = 0
	updTask.Status = models.InProgressTaskStatus

	err = s.db.UpdateTask(ctx, updTask)
	if err != nil {
		return models.Task{}, err
	}

	// Отправляем письмо об обновлении задачи
	err = s.mail.SendResultMail(ctx, mailResult)
	if err != nil {
		return models.Task{}, err
	}

	// Генерируем ссылки и отправляем письмо 1-ому согласующему
	links := generateLink(ctx, updTask)
	mailApproval := models.MailToApproval{
		Destination: updTask.ApprovalLogins[updTask.CurrentApprovalNumber],
		ApproveLink: links.ApprovalLink,
		DeclineLink: links.DeclineLink,
	}

	err = s.mail.SendApprovalMail(ctx, mailApproval)
	if err != nil {
		return models.Task{}, err
	}

	return updTask, nil
}

// DeleteTask удалить задачу
func (s *Service) DeleteTask(ctx context.Context, taskID, userLogin string) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "DeleteTask")
	defer span.End()

	// Находим task по ID
	task, err := s.db.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Проверяем автора задачи
	if task.InitiatorLogin != userLogin {
		return taskErrors.ErrInitiatorInvalid
	}

	// Готовим шаблон письма об удалении задачи
	mailResult := models.ResultMail{
		Destinations: task.ApprovalLogins,
		TaskID:       task.ID,
		Result:       "task was deleted",
	}

	// Удаляем задачу
	err = s.db.DeleteTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Отправляем письмо об обновлении задачи
	err = s.mail.SendResultMail(ctx, mailResult)
	if err != nil {
		return err
	}

	return nil
}

// GetTask получить задачу по ID
func (s *Service) GetTaskByID(ctx context.Context, taskID string) (models.Task, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "GetTaskByID")
	defer span.End()

	// Находим task по ID
	task, err := s.db.GetTask(ctx, taskID)
	if err != nil {
		return models.Task{}, fmt.Errorf("get task was failed: %w", err)
	}

	return task, nil
}

// GetAllTasks получить все задачи
func (s *Service) GetAllTasks(ctx context.Context) ([]models.Task, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "GetAllTasks")
	defer span.End()

	return s.db.GetTasks(ctx)
}

// ApprovalTask согласовать задачу текущим согласующим
func (s *Service) ApprovalTask(ctx context.Context, taskID, userLogin string) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "ApprovalTask")
	defer span.End()

	// Находим task по ID
	task, err := s.db.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Проверяем, что задача в процессе согласования
	if task.Status != models.InProgressTaskStatus {
		return taskErrors.ErrTaskStatusInvalid
	}

	// Проверяем текущего согласующего в запросе и в БД
	if task.ApprovalLogins[task.CurrentApprovalNumber] != userLogin {
		return taskErrors.ErrApprovalInvalid
	}

	// Отправляем информацию о нажатии на ссылку текущим согласующим в Analytics
	err = s.analytica.SendTimestamp(ctx, models.TimestampData{
		TaskID:    task.ID,
		Approver:  task.ApprovalLogins[task.CurrentApprovalNumber],
		EventType: models.ApproveTypeEvent,
		End:       time.Now().UTC(),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	var mail interface{}

	if task.CurrentApprovalNumber < len(task.ApprovalLogins)-1 {
		// Если согласование не закончено, изменяем счётчик согласующего
		task.CurrentApprovalNumber++
		links := generateLink(ctx, task)
		mail = models.MailToApproval{
			Destination: task.ApprovalLogins[task.CurrentApprovalNumber],
			ApproveLink: links.ApprovalLink,
			DeclineLink: links.DeclineLink,
		}
	} else {
		// Если согласование закончено, то меняем статус задачи
		task.Status = models.ApprovedTaskStatus
		mail = models.ResultMail{
			Destinations: task.ApprovalLogins,
			TaskID:       task.ID,
			Result:       "task was approved",
		}
	}

	// Обновляем задачу в БД
	err = s.db.UpdateTask(ctx, task)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	// Отправляем соответствующее письмо
	if task.Status == models.ApprovedTaskStatus {
		// Отправляем информацию о cогласовании задачи в Analytics
		err = s.analytica.SendResult(ctx, models.ResultData{
			TaskID: taskID,
			Result: true,
		})
		if err != nil {
			span.SetStatus(codes.Error, err.Error())

			return err
		}

		err = s.mail.SendResultMail(ctx, mail.(models.ResultMail))
		if err != nil {
			span.SetStatus(codes.Error, err.Error())

			return err
		}
		// Отправляем информацию об отправке письма о завершении согласования задачи в Analytics
		err = s.analytica.SendTimestamp(ctx, models.TimestampData{
			TaskID:    task.ID,
			EventType: models.TaskTypeEvent,
			End:       time.Now().UTC(),
		})
	} else {
		err = s.mail.SendApprovalMail(ctx, mail.(models.MailToApproval))
		if err != nil {
			span.SetStatus(codes.Error, err.Error())

			return err
		}
		// Отправляем информацию об отправке письма текущему согласующему в Analytics
		err = s.analytica.SendTimestamp(ctx, models.TimestampData{
			TaskID:    task.ID,
			Approver:  task.ApprovalLogins[task.CurrentApprovalNumber],
			EventType: models.ApproveTypeEvent,
			Start:     time.Now().UTC(),
		})
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

// DeclineTask отклонить задачу текущим согласующим
func (s *Service) DeclineTask(ctx context.Context, taskID, userLogin string) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "DeclineTask")
	defer span.End()

	// Находим task по ID
	task, err := s.db.GetTask(ctx, taskID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	// Проверяем, что задача в процессе согласования
	if task.Status != models.InProgressTaskStatus {
		return taskErrors.ErrTaskStatusInvalid
	}

	// Проверяем текущего согласующего в запросе и в БД
	if task.ApprovalLogins[task.CurrentApprovalNumber] != userLogin {
		return taskErrors.ErrApprovalInvalid
	}

	// Отправляем информацию о нажатии на ссылку текущим согласующим в Analytics
	err = s.analytica.SendTimestamp(ctx, models.TimestampData{
		TaskID:    task.ID,
		Approver:  task.ApprovalLogins[task.CurrentApprovalNumber],
		EventType: models.ApproveTypeEvent,
		End:       time.Now().UTC(),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	// Обновляем задачу в БД
	task.Status = models.DeclinedTaskStatus

	err = s.db.UpdateTask(ctx, task)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	// Отправляем информацию о несогласовании задачи в Analytics
	err = s.analytica.SendResult(ctx, models.ResultData{
		TaskID: taskID,
		Result: false,
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	// Отправляем письмо
	mail := models.ResultMail{
		Destinations: task.ApprovalLogins,
		TaskID:       task.ID,
		Result:       "task was declined",
	}

	err = s.mail.SendResultMail(ctx, mail)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	// Отправляем информацию об отправке письма о завершении согласования задачи в Analytics
	err = s.analytica.SendTimestamp(ctx, models.TimestampData{
		TaskID:    task.ID,
		EventType: models.TaskTypeEvent,
		End:       time.Now().UTC(),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

func generateLink(ctx context.Context, task models.Task) models.Links {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "generateLink")
	defer span.End()

	host := http.HostFromContext(ctx)
	approveLink := host + "/tasks/" + task.ID + "/approve/" + task.ApprovalLogins[task.CurrentApprovalNumber]
	declineLink := host + "/tasks/" + task.ID + "/decline/" + task.ApprovalLogins[task.CurrentApprovalNumber]

	return models.Links{
		ApprovalLink: approveLink,
		DeclineLink:  declineLink,
	}
}
