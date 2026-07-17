package internal

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
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

		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}

func AccessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		requestID, _ := c.Get("request_id")
		requestIDStr, _ := requestID.(string)

		logger.InfoContext(c.Request.Context(), "request started",
			slog.String("request_id", requestIDStr),
			slog.String("method", c.Request.Method),
			slog.String("url", c.Request.URL.String()),
		)

		defer func() {
			attrs := []any{
				slog.String("request_id", requestIDStr),
				slog.String("method", c.Request.Method),
				slog.String("url", c.Request.URL.String()),
				slog.Int("status_code", c.Writer.Status()),
				slog.Duration("latency", time.Since(startedAt)),
			}

			// uid と player_id は認証・バリデーションの各ミドルウェアが設定するため、
			// c.Next() 完了後であるこのdefer内でのみ参照できる。設定されないエンドポイント
			// では出力しない。
			if uid := helper.GetUID(c); uid != "" {
				attrs = append(attrs, slog.String("uid", uid))
			}

			if playerId := helper.GetPlayerId(c); playerId != "" {
				attrs = append(attrs, slog.String("player_id", playerId))
			}

			logger.InfoContext(c.Request.Context(), "request finished", attrs...)
		}()

		c.Next()
	}
}
