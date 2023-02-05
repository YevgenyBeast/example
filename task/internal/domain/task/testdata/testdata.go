package testdata

import "task/internal/domain/models"

var (
	Host = "testhost"
	Task = models.Task{
		ID:                    "53f891df-9789-4a77-9f80-23176384c94c",
		InitiatorLogin:        "author1",
		ApprovalLogins:        []string{"approval1", "approval2"},
		CurrentApprovalNumber: 0,
		Status:                models.InProgressTaskStatus,
	}
	OldTask = models.Task{
		ID:                    "53f891df-9789-4a77-9f80-23176384c94c",
		InitiatorLogin:        "author1",
		ApprovalLogins:        []string{"approval1", "approval2", "approval3"},
		CurrentApprovalNumber: 1,
		Status:                models.DeclinedTaskStatus,
	}
	TaskApproveStep = models.Task{
		ID:                    "53f891df-9789-4a77-9f80-23176384c94c",
		InitiatorLogin:        "author1",
		ApprovalLogins:        []string{"approval1", "approval2"},
		CurrentApprovalNumber: 1,
		Status:                models.InProgressTaskStatus,
	}
	TaskDeclineStep = models.Task{
		ID:                    "53f891df-9789-4a77-9f80-23176384c94c",
		InitiatorLogin:        "author1",
		ApprovalLogins:        []string{"approval1", "approval2"},
		CurrentApprovalNumber: 0,
		Status:                models.DeclinedTaskStatus,
	}
	CreateMail = models.MailToApproval{
		Destination: Task.ApprovalLogins[0],
		ApproveLink: Host + "/tasks/" + Task.ID + "/approve/" + Task.ApprovalLogins[0],
		DeclineLink: Host + "/tasks/" + Task.ID + "/decline/" + Task.ApprovalLogins[0],
	}
	UpdateMail = models.ResultMail{
		Destinations: OldTask.ApprovalLogins,
		TaskID:       OldTask.ID,
		Result:       "task was updated",
	}
	DeleteMail = models.ResultMail{
		Destinations: Task.ApprovalLogins,
		TaskID:       Task.ID,
		Result:       "task was deleted",
	}
	ApproveMail = models.MailToApproval{
		Destination: Task.ApprovalLogins[1],
		ApproveLink: Host + "/tasks/" + Task.ID + "/approve/" + Task.ApprovalLogins[1],
		DeclineLink: Host + "/tasks/" + Task.ID + "/decline/" + Task.ApprovalLogins[1],
	}
	DeclineMail = models.ResultMail{
		Destinations: Task.ApprovalLogins,
		TaskID:       Task.ID,
		Result:       "task was declined",
	}
)
