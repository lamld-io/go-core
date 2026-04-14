package apperror

import (
	"fmt"
	"net/http"
)

// Code định nghĩa mã lỗi ứng dụng.
type Code string

const (
	CodeBadRequest     Code = "BAD_REQUEST"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeNotFound       Code = "NOT_FOUND"
	CodeConflict       Code = "CONFLICT"
	CodeInternalError  Code = "INTERNAL_ERROR"
	CodeValidation     Code = "VALIDATION_ERROR"
	CodeTokenExpired   Code = "TOKEN_EXPIRED"
	CodeTokenInvalid   Code = "TOKEN_INVALID"
	CodeRateLimited    Code = "RATE_LIMITED"
	CodeServiceUnavail Code = "SERVICE_UNAVAILABLE"
)

// AppError là lỗi ứng dụng có cấu trúc, chứa mã lỗi và message.
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus trả về HTTP status code tương ứng với mã lỗi.
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case CodeBadRequest, CodeValidation:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeTokenExpired, CodeTokenInvalid:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeServiceUnavail:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Constructor helpers

func BadRequest(msg string) *AppError {
	return &AppError{Code: CodeBadRequest, Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg}
}

func NotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg}
}

func InternalError(msg string, err error) *AppError {
	return &AppError{Code: CodeInternalError, Message: msg, Err: err}
}

func ValidationError(msg string) *AppError {
	return &AppError{Code: CodeValidation, Message: msg}
}

func TokenExpired() *AppError {
	return &AppError{Code: CodeTokenExpired, Message: "token has expired"}
}

func TokenInvalid() *AppError {
	return &AppError{Code: CodeTokenInvalid, Message: "token is invalid"}
}

func RateLimited() *AppError {
	return &AppError{Code: CodeRateLimited, Message: "rate limit exceeded"}
}

func ServiceUnavailable(msg string) *AppError {
	return &AppError{Code: CodeServiceUnavail, Message: msg}
}
