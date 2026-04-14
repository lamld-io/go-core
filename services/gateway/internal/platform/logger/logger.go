package logger

import (
	"log/slog"
	"os"
)

// Setup khởi tạo structured logger (slog) cho Gateway Service.
func Setup(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	switch env {
	case "production", "prod":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
