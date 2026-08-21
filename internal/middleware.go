package internal

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

// MaxRequestBodyBytes はリクエストボディとして受け付ける最大サイズ。
// メモ欄などを含めても十分な余裕がある一方、巨大なJSONによってメモリを
// 圧迫されることを防げる大きさとして 1MiB を採用している。
const MaxRequestBodyBytes = 1 << 20

// BodySizeLimitMiddleware はリクエストボディをmaxBytesまでに制限する。
//
// 上限が無い場合、ShouldBindJSONはボディ全体をメモリへ読み込むため、
// 巨大なボディを送りつけるだけでメモリを枯渇させられる。上限を超えたボディは
// 読み取り時にエラーとなり、各バリデーションミドルウェアが400を返す。
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewString()

		c.Set("request_id", requestID)

		// request_id を Request の context にも載せる。controller が
		// ctx.Request.Context() を下層へ渡すことで、usecase / infrastructure の
		// ログにも同じ request_id が付き、1リクエストを層をまたいで追える。
		c.Request = c.Request.WithContext(
			logging.ContextWithRequestID(c.Request.Context(), requestID),
		)

		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}

func AccessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		// request_id と uid は ContextHandler が context から自動付与するため、
		// ここでは明示的に渡さない(渡すとJSONのキーが重複する)。
		logger.InfoContext(c.Request.Context(), "request started",
			slog.String("method", c.Request.Method),
			slog.String("url", c.Request.URL.String()),
		)

		defer func() {
			attrs := []any{
				slog.String("method", c.Request.Method),
				slog.String("url", c.Request.URL.String()),
				slog.Int("status_code", c.Writer.Status()),
				slog.Duration("latency", time.Since(startedAt)),
			}

			// player_id はバリデーションミドルウェアが設定するため、c.Next() 完了後で
			// あるこのdefer内でのみ参照できる。設定されないエンドポイントでは出力しない。
			if playerId := helper.GetPlayerId(c); playerId != "" {
				attrs = append(attrs, slog.String("player_id", playerId))
			}

			logger.InfoContext(c.Request.Context(), "request finished", attrs...)
		}()

		c.Next()
	}
}

// RecoveryMiddleware は panic を捕捉して 500 を返す。
//
// gin.Recovery() はスタックトレースを独自のテキスト形式で stderr へ書くため、
// 他のログと同じJSONとして集約・検索できない。gin側の書き出し先を nil にして
// その出力を止め、slog で JSON として出し直している。
// 応答は gin.Recovery() と同じくボディ無しの 500 のまま。
//
// なお書き出し先を nil にしたことで、クライアント切断(broken pipe)時に
// gin が出していたログも出なくなるが、これはクライアント都合の切断であり
// サーバ側で対処するものではない。
func RecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		logger.ErrorContext(c.Request.Context(), "panic recovered",
			slog.String("method", c.Request.Method),
			slog.String("url", c.Request.URL.String()),
			slog.Any("panic", recovered),
			// recover 済みでも、この defer はpanicしたフレームの上で動くため
			// スタックにはpanic発生箇所が含まれる。
			slog.String("stack", string(debug.Stack())),
		)

		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
