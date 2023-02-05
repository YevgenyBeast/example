package models

// MailToApproval описывает стурктуру письма для согласующим
type MailToApproval struct {
	Destination string
	ApproveLink string
	DeclineLink string
}

// ResultMail описывает структуру письма с конечным результатом согласования задачи
type ResultMail struct {
	Destinations []string
	TaskID       string
	Result       string
}
