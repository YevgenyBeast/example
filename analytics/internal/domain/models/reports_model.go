package models

// ResultsReport данные с количеством согласованных/несогласованных задач
type ResultsReport struct {
	ApprovedTasks int `json:"approvedtasks"`
	DeclinedTasks int `json:"declinedtasks"`
}

// TimeReport отчёт с данными о времени согласования задачи
type TimeReport struct {
	TaskID      string `json:"taskid"`
	ApproveTime string `json:"approvetime"`
	TotalTime   string `json:"totaltime"`
}
