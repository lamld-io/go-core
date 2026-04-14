package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository định nghĩa interface truy cập dữ liệu User.
// Tầng domain chỉ khai báo interface, implementation nằm ở tầng repository.
type UserRepository interface {
	// Create tạo user mới. Trả về ErrUserAlreadyExists nếu email đã tồn tại.
	Create(ctx context.Context, user *User) error

	// GetByID tìm user theo ID. Trả về ErrUserNotFound nếu không tồn tại.
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByEmail tìm user theo email. Trả về ErrUserNotFound nếu không tồn tại.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update cập nhật thông tin user.
	Update(ctx context.Context, user *User) error
}

// ActionTokenRepository định nghĩa truy cập dữ liệu token nghiệp vụ.
type ActionTokenRepository interface {
	Create(ctx context.Context, token *ActionToken) error
	GetUsableByTokenHash(ctx context.Context, tokenHash string, tokenType ActionTokenType) (*ActionToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error
	RevokeByUserAndType(ctx context.Context, userID uuid.UUID, tokenType ActionTokenType) error
	DeleteExpired(ctx context.Context) error
}

// TokenRepository định nghĩa interface truy cập dữ liệu RefreshToken.
type TokenRepository interface {
	// Create lưu refresh token mới.
	Create(ctx context.Context, token *RefreshToken) error

	// GetByTokenHash tìm refresh token theo hash. Trả về ErrTokenNotFound nếu không tồn tại.
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// ListByUserID lấy danh sách tất cả refresh token của một user (chưa bị thu hồi).
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)

	// RevokeByUserID thu hồi tất cả refresh token của một user.
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error

	// RevokeByIDAndUserID thu hồi một refresh token cụ thể của user. Ngăn chặn IDOR.
	RevokeByIDAndUserID(ctx context.Context, id, userID uuid.UUID) error

	// DeleteExpired xoá các token đã hết hạn (dùng cho cleanup job).
	DeleteExpired(ctx context.Context) error
}

// LoginLockoutPolicyRepository định nghĩa truy cập policy khoá tài khoản.
type LoginLockoutPolicyRepository interface {
	Get(ctx context.Context) (*LoginLockoutPolicy, error)
	Upsert(ctx context.Context, policy *LoginLockoutPolicy) error
}
