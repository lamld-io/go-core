package logger

import (
	"log/slog"
	"os"
)

// Setup khởi tạo structured logger (slog) cho toàn bộ service.
// Log format: JSON khi production, Text khi development.
func Setup(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	switch env {
	case "production", "prod":
		// JSON format cho production — dễ parse bởi log aggregator (ELK, Loki, v.v.)
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		// Text format cho development — dễ đọc trong terminal.
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
