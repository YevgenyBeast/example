package models

import "time"

// ResultData структура данных с резултатами согласования задачи
type ResultData struct {
	TaskID string `json:"taskID"`
	Result bool   `json:"result"`
}

// TimestampData структура данных с временными метками событий по задаче
type TimestampData struct {
	TaskID    string    `json:"taskID"`
	Approver  string    `json:"approver,omitempty"`
	EventType Event     `json:"eventType"`
	Start     time.Time `json:"start,omitempty"`
	End       time.Time `json:"end,omitempty"`
}

// Event содержит информации о событии: согласование или создание/закрытие задачи
type Event string

const (
	TaskTypeEvent    Event = "task"
	ApproveTypeEvent Event = "approve"
)
