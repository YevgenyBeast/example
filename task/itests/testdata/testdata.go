package testdata

import "task/internal/domain/models"

var (
	Task1 = models.Task{
		InitiatorLogin: "author1",
		ApprovalLogins: []string{"approval1", "approval2", "approval3"},
	}
	Task1Update = models.Task{
		ApprovalLogins: []string{"approval1", "approval3"},
	}
	Task2 = models.Task{
		InitiatorLogin: "author2",
		ApprovalLogins: []string{"approval1", "approval2"},
	}
	AccessToken  string = "realAccessToken"
	RefreshToken string = "realRefreshToken"
)
