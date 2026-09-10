package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Config は InitLogger へ渡す設定。
type Config struct {
	// Level は "DEBUG" / "INFO" / "WARN" / "ERROR"。未知の値は INFO になる。
	Level string
	// AppName は全レコードへ付く appname 属性。ログを出したプロセスを識別する
	// (APIサーバと cron の各バッチが同じ集約先へ出るため)。
	AppName string
}

// InitLogger は全プロセス共通のロガーを作る。呼び出し側が slog.SetDefault で
// 既定のロガーに設定することを前提にしている。
//
// APIサーバだけでなく cmd 配下のバッチもこれを使う。バッチが slog の既定ハンドラの
// まま動くと、同じ usecase / infrastructure のログがテキスト形式で出て layer や
// ソース位置も落ちるため、ログの追い方がプロセスごとに変わってしまう。
func InitLogger(config Config) *slog.Logger {
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
	handler := NewContextHandler(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
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
