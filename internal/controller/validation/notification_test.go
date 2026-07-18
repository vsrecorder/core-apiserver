package validation

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestNotificationGetMiddleware(t *testing.T) {
	t.Run("正常系_limit指定をコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "limit=30")

		NotificationGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 30, helper.GetLimit(ctx))
	})

	t.Run("正常系_limit未指定ならデフォルト値を設定して通過する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "")

		NotificationGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, helper.DefaultLimit, helper.GetLimit(ctx))
	})

	t.Run("異常系_limitが数値でなければ400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "limit=abc")

		NotificationGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
