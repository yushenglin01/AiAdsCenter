package apperror

import "fmt"

type Error struct {
	Code       int
	HTTPStatus int
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

func New(code, status int, message string) *Error {
	return &Error{Code: code, HTTPStatus: status, Message: message}
}

func Validation(message string) *Error { return New(10001, 400, message) }

var (
	InvalidArgument = New(10001, 400, "invalid argument")
	Unauthorized    = New(10002, 401, "authentication required")
	Forbidden       = New(10003, 403, "permission denied")
	NotFound        = New(10004, 404, "resource not found")
	Conflict        = New(10005, 409, "resource conflict")
	Internal        = New(10500, 500, "internal server error")
)
