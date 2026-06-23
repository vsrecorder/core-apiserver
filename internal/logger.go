package internal

import (
	"log/slog"
	"os"
	"strings"
)

type LogConfig struct {
	Level   string
	AppName string
}

func InitLogger(config LogConfig) *slog.Logger {
	var level slog.Level

	switch strings.ToUpper(config.Level) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler).With(
		slog.String("appname", config.AppName),
	)

	return logger
}
