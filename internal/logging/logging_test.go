package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// newTestLogger は出力を JSON として検証できるロガーを返す。
func newTestLogger(t *testing.T, addSource bool) (*slog.Logger, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}
	handler := NewContextHandler(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: addSource,
	}))

	return slog.New(handler), buf
}

// decode は出力された最後の1レコードを map として返す。
func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.NotEmpty(t, lines)

	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &record))

	return record
}

func TestContextHandler(t *testing.T) {
	t.Run("正常系_contextのrequest_idとuidが自動で付与される", func(t *testing.T) {
		logger, buf := newTestLogger(t, false)

		ctx := ContextWithRequestID(context.Background(), "req-1")
		ctx = ContextWithUID(ctx, "uid-1")

		logger.ErrorContext(ctx, "something failed")

		record := decode(t, buf)
		require.Equal(t, "req-1", record["request_id"])
		require.Equal(t, "uid-1", record["uid"])
	})

	t.Run("正常系_contextに値が無ければ属性を付与しない", func(t *testing.T) {
		logger, buf := newTestLogger(t, false)

		logger.ErrorContext(context.Background(), "something failed")

		record := decode(t, buf)
		require.NotContains(t, record, "request_id")
		require.NotContains(t, record, "uid")
	})

	t.Run("正常系_Withで属性を足してもrequest_idの付与が維持される", func(t *testing.T) {
		logger, buf := newTestLogger(t, false)

		ctx := ContextWithRequestID(context.Background(), "req-2")

		// WithAttrs がラップを外していると、ここで request_id が消える。
		logger.With(slog.String("layer", "usecase")).ErrorContext(ctx, "something failed")

		record := decode(t, buf)
		require.Equal(t, "req-2", record["request_id"])
		require.Equal(t, "usecase", record["layer"])
	})

	t.Run("正常系_nilのcontextでも取得関数が空文字を返す", func(t *testing.T) {
		//nolint:staticcheck // nil を渡しても落ちないことの確認
		require.Equal(t, "", RequestIDFromContext(nil))
		require.Equal(t, "", UIDFromContext(nil))
	})
}

func TestLogAt(t *testing.T) {
	t.Run("正常系_ヘルパーではなく呼び出し元の位置が記録される", func(t *testing.T) {
		logger, buf := newTestLogger(t, true)

		logViaHelper(context.Background(), logger, "failed")

		record := decode(t, buf)

		source, ok := record["source"].(map[string]any)
		require.True(t, ok)

		// skip=1 を渡しているため、logViaHelper ではなく その呼び出し元 が記録される。
		require.Contains(t, source["function"], "TestLogAt")
		require.Contains(t, source["file"], "logging_test.go")
	})

	t.Run("正常系_出力レベルより低いログは出力されない", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := slog.New(NewContextHandler(slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level: slog.LevelError,
		})))

		LogAt(context.Background(), logger, slog.LevelDebug, 0, "debug message")

		require.Empty(t, buf.String())
	})
}

// logViaHelper は LogAt を内部で呼ぶヘルパー。ソース位置の skip 検証に使う。
func logViaHelper(ctx context.Context, logger *slog.Logger, msg string) {
	LogAt(ctx, logger, slog.LevelError, 1, msg)
}

func TestCallerOperation(t *testing.T) {
	t.Run("正常系_呼び出し元の関数名をパッケージ名付きで返す", func(t *testing.T) {
		name := CallerOperation(0)

		require.Equal(t, "logging.TestCallerOperation.func1", name)
	})

	t.Run("正常系_skipで呼び出し元をさかのぼれる", func(t *testing.T) {
		require.Equal(t, "logging.TestCallerOperation.func2", operationViaHelper())
	})
}

func operationViaHelper() string {
	return CallerOperation(1)
}

func TestErr(t *testing.T) {
	t.Run("正常系_エラーメッセージをerror_messageへ載せる", func(t *testing.T) {
		attr := Err(context.DeadlineExceeded)

		require.Equal(t, "error_message", attr.Key)
		require.Equal(t, context.DeadlineExceeded.Error(), attr.Value.String())
	})

	t.Run("正常系_nilを渡しても落ちない", func(t *testing.T) {
		require.Equal(t, "", Err(nil).Value.String())
	})
}

func TestLayered(t *testing.T) {
	t.Run("正常系_layer属性が付いたロガーを返す", func(t *testing.T) {
		buf := &bytes.Buffer{}
		prev := slog.Default()
		t.Cleanup(func() { slog.SetDefault(prev) })

		slog.SetDefault(slog.New(NewContextHandler(slog.NewJSONHandler(buf, nil))))

		Layered(LayerInfrastructure).ErrorContext(context.Background(), "failed")

		require.Equal(t, LayerInfrastructure, decode(t, buf)["layer"])
	})
}
