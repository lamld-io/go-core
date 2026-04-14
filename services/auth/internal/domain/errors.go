package domain

import (
	"fmt"
	"time"

	"github.com/base-go/base/pkg/apperror"
)

// Domain-specific errors cho Auth Service.
// Các lỗi này được định nghĩa ở tầng domain, không phụ thuộc HTTP hay framework.

var (
	ErrUserNotFound       = apperror.NotFound("user not found")
	ErrUserAlreadyExists  = apperror.Conflict("user with this email already exists")
	ErrInvalidCredentials = apperror.Unauthorized("invalid email or password")
	ErrUserInactive       = apperror.Forbidden("user account is inactive")
	ErrEmailNotVerified   = apperror.Forbidden("email address is not verified")

	ErrTokenNotFound = apperror.NotFound("refresh token not found")
	ErrTokenRevoked  = apperror.Unauthorized("refresh token has been revoked")
	ErrTokenExpired  = apperror.TokenExpired()
	ErrTokenInvalid  = apperror.TokenInvalid()

	ErrActionTokenNotFound        = apperror.NotFound("action token not found")
	ErrActionTokenInvalid         = apperror.BadRequest("action token is invalid or expired")
	ErrLoginLockoutPolicyNotFound = apperror.NotFound("login lockout policy not found")

	ErrInvalidRole = apperror.BadRequest("invalid role")
	ErrForbidden   = apperror.Forbidden("you do not have permission to perform this action")

	ErrWeakPassword = apperror.ValidationError("password does not meet the required policy")
)

// NewAccountLockedError trả về lỗi tài khoản đang bị khoá tạm thời.
func NewAccountLockedError(lockedUntil time.Time) error {
	return apperror.Forbidden(
		fmt.Sprintf("account is locked until %s", lockedUntil.UTC().Format(time.RFC3339)),
	)
}
