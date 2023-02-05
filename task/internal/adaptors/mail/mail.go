package mail

import (
	"context"
	"encoding/json"
	"fmt"
	"task/internal/adaptors/http"
	"task/internal/domain/models"

	"go.opentelemetry.io/otel"
)

type Adaptor struct{}

// New конструктор адаптора Mail
func New() *Adaptor {
	return &Adaptor{}
}

// SendApprovalMail отправка письма согласования
func (m *Adaptor) SendApprovalMail(ctx context.Context, mail models.MailToApproval) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SendApprovalMail")
	defer span.End()

	logger := http.LoggerFromContext(ctx)

	mailJSON, err := json.Marshal(mail)
	if err != nil {
		return fmt.Errorf("send mail was failed: %s", err.Error())
	}

	logger.Info("Send approval-mail: ", string(mailJSON))

	return nil
}

// SendResultMail отправка письма с результатом
func (m *Adaptor) SendResultMail(ctx context.Context, mail models.ResultMail) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SendResultMail")
	defer span.End()

	logger := http.LoggerFromContext(ctx)

	mailJSON, err := json.Marshal(mail)
	if err != nil {
		return fmt.Errorf("send mail was failed: %s", err.Error())
	}

	logger.Info("Send result-mail: ", string(mailJSON))

	return nil
}
