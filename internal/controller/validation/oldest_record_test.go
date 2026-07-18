package validation

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestOldestRecordGetMiddleware(t *testing.T) {
	t.Run("正常系_deck_id指定をコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "deck_id=01HD7Y3K8D6FDHMHTZ2GT41TN2")

		OldestRecordGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", helper.GetDeckId(ctx))
	})

	t.Run("正常系_deck_id未指定なら空文字を設定して通過する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "")

		OldestRecordGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, helper.GetDeckId(ctx))
	})
}
