package internal

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

// decodeRecords は出力された全ログレコードを map のスライスへ変換する。
func decodeRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}

		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		records = append(records, record)
	}

	return records
}

// newTestLogger は InitLogger と同じ構成(ContextHandler 付き)のロガーを返す。
func newTestLogger(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}
	handler := logging.NewContextHandler(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return slog.New(handler), buf
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("正常系_request_idがアクセスログと下位層のログで一致する", func(t *testing.T) {
		logger, buf := newTestLogger(t)

		prev := slog.Default()
		t.Cleanup(func() { slog.SetDefault(prev) })
		slog.SetDefault(logger)

		r := gin.New()
		r.Use(RequestIDMiddleware(), AccessLogMiddleware(logger))
		r.GET("/", func(ctx *gin.Context) {
			// usecase / infrastructure 層が ctx を受け取って出すログの再現。
			logging.Layered(logging.LayerUsecase).
				ErrorContext(ctx.Request.Context(), "usecase failed")
			logging.Layered(logging.LayerInfrastructure).
				ErrorContext(ctx.Request.Context(), "repository operation failed")

			ctx.JSON(http.StatusOK, gin.H{})
		})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		requestID := w.Header().Get("X-Request-ID")
		require.NotEmpty(t, requestID)

		records := decodeRecords(t, buf)
		// request started / usecase / infrastructure / request finished の4件。
		require.Len(t, records, 4)

		for _, record := range records {
			require.Equal(t, requestID, record["request_id"], record["msg"])
		}

		require.Equal(t, logging.LayerUsecase, records[1]["layer"])
		require.Equal(t, logging.LayerInfrastructure, records[2]["layer"])
	})

	t.Run("正常系_認証後に設定したuidが下位層のログにも載る", func(t *testing.T) {
		logger, buf := newTestLogger(t)

		prev := slog.Default()
		t.Cleanup(func() { slog.SetDefault(prev) })
		slog.SetDefault(logger)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		r := gin.New()
		r.Use(RequestIDMiddleware(), AccessLogMiddleware(logger))
		r.GET("/",
			// 認証ミドルウェア相当。
			func(ctx *gin.Context) { helper.SetUID(ctx, uid) },
			func(ctx *gin.Context) {
				logging.Layered(logging.LayerUsecase).
					ErrorContext(ctx.Request.Context(), "usecase failed")

				ctx.JSON(http.StatusOK, gin.H{})
			},
		)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		records := decodeRecords(t, buf)
		require.Len(t, records, 3)

		// "request started" は認証より前に出るため uid を持たない。
		require.NotContains(t, records[0], "uid")
		require.Equal(t, uid, records[1]["uid"])
		require.Equal(t, uid, records[2]["uid"])
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("正常系_panicをJSONログへ出して500を返す", func(t *testing.T) {
		logger, buf := newTestLogger(t)

		r := gin.New()
		r.Use(RequestIDMiddleware(), RecoveryMiddleware(logger))
		r.GET("/", func(ctx *gin.Context) {
			panic("something went wrong")
		})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusInternalServerError, w.Code)

		records := decodeRecords(t, buf)
		require.Len(t, records, 1)

		record := records[0]
		require.Equal(t, "panic recovered", record["msg"])
		require.Equal(t, "ERROR", record["level"])
		require.Equal(t, "something went wrong", record["panic"])
		require.Equal(t, w.Header().Get("X-Request-ID"), record["request_id"])
		// recover 後でもpanic発生箇所がスタックに残っていること。
		require.Contains(t, record["stack"], "TestRecoveryMiddleware")
	})
}
