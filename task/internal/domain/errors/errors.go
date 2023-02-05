package errors

import "errors"

var (
	ErrInitiatorInvalid  = errors.New("invalid initiator login")
	ErrApprovalInvalid   = errors.New("invalid approval login")
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskStatusInvalid = errors.New("task is not in progress")
)
