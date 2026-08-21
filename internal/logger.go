package internal

import (
	"log/slog"
	"os"
	"strings"

	"github.com/vsrecorder/core-apiserver/internal/logging"
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

	// ContextHandler で包むことで、context に載った request_id / uid が
	// 全レイヤーのログへ自動的に付与される。
	handler := logging.NewContextHandler(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		// 各層のログにファイル:行が入ると、同じメッセージが複数箇所から出ていても
		// 発生箇所を一意に特定できる。
		AddSource: true,
	}))

	logger := slog.New(handler).With(
		slog.String("appname", config.AppName),
	)

	return logger
}
