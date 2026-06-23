package internal

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
			logger.InfoContext(c.Request.Context(), "request finished",
				slog.String("request_id", requestIDStr),
				slog.String("method", c.Request.Method),
				slog.String("url", c.Request.URL.String()),
				slog.Int("status_code", c.Writer.Status()),
				slog.Duration("latency", time.Since(startedAt)),
			)
		}()

		c.Next()
	}
}
