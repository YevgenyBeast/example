package ports

import (
	"context"
	"task/internal/domain/models"
)

// Интерфейс для работы с письмами
type Mail interface {
	SendApprovalMail(ctx context.Context, mail models.MailToApproval) error
	SendResultMail(ctx context.Context, mail models.ResultMail) error
}
