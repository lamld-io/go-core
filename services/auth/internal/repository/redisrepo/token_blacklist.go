package redisrepo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type tokenBlacklist struct {
	client *redis.Client
}

// NewTokenBlacklist tạo repository quản lý token blacklist trong Redis.
func NewTokenBlacklist(client *redis.Client) *tokenBlacklist {
	return &tokenBlacklist{client: client}
}

// BlacklistToken thêm token ID vào blacklist với thời gian tồn tại bằng TTL của token.
func (r *tokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	if r.client == nil {
		slog.WarnContext(ctx, "redis is not connected, skip blacklisting token")
		return nil
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // Token already expired naturally
	}

	key := fmt.Sprintf("blacklist:token:%s", tokenID)
	
	if err := r.client.Set(ctx, key, "revoked", ttl).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to blacklist token", "error", err, "token_id", tokenID)
		return err
	}
	
	slog.DebugContext(ctx, "token added to blacklist", "token_id", tokenID, "ttl", ttl)
	return nil
}

// IsBlacklisted kiểm tra xem token ID đã bị blacklist hay chưa.
func (r *tokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	if r.client == nil {
		return false, nil // Fail-open: don't block user if redis is down
	}

	key := fmt.Sprintf("blacklist:token:%s", tokenID)
	err := r.client.Get(ctx, key).Err()
	
	if err == redis.Nil {
		return false, nil // Not found = not blacklisted
	} else if err != nil {
		slog.ErrorContext(ctx, "failed to check token blacklist", "error", err, "token_id", tokenID)
		return false, nil // Fail-open: bypass check if redis throws an error
	}

	return true, nil
}
