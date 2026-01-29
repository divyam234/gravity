package errors

import "fmt"

type ErrorCode string

const (
	CodeNotFound          ErrorCode = "NOT_FOUND"
	CodeValidationFailed  ErrorCode = "VALIDATION_FAILED"
	CodeInvalidTransition ErrorCode = "INVALID_TRANSITION"
	CodeInternalError     ErrorCode = "INTERNAL_ERROR"
	CodeInvalidOperation  ErrorCode = "INVALID_OPERATION"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

type NotFoundError struct {
	ID     string
	Entity string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %s not found", e.Entity, e.ID)
}

func NewNotFound(entity, id string) *NotFoundError {
	return &NotFoundError{ID: id, Entity: entity}
}
