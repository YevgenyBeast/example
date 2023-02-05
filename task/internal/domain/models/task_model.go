package models

// Task стурктура с данными о задаче
type Task struct {
	ID                    string     `json:"id,omitempty" bson:"_id" example:"55e1b4bf-12a7-4809-b0f5-c17e1f69e7fa"`
	InitiatorLogin        string     `json:"initiatorLogin" bson:"initiatorLogin" example:"author"`
	ApprovalLogins        []string   `json:"approvalLogins" bson:"approvalsLogins" example:"approval1,approval2"`
	CurrentApprovalNumber int        `json:"currentApprovalNumber" bson:"currentApprovalNumber" example:"0"`
	Status                TaskStatus `json:"status" bson:"status" example:"0"`
}

// TaskRes структура задачи для ответа на запрос
type TaskRes struct {
	ID             string     `json:"id" example:"55e1b4bf-12a7-4809-b0f5-c17e1f69e7fa"`
	InitiatorLogin string     `json:"initiatorLogin" example:"author"`
	Approval       []Approval `json:"approval"`
}

// Approval структура данных о согласователе
type Approval struct {
	Login string `json:"login" example:"approval1, approval2"`
}

// TaskStatus тип для статуса задачи
type TaskStatus int8

const (
	// InProgressTaskStatus - задача в процессе согласования
	InProgressTaskStatus TaskStatus = iota
	// ApprovedTaskStatus - задача согласована
	ApprovedTaskStatus
	// DeclinedTaskStatus - задача не согласована
	DeclinedTaskStatus
)
