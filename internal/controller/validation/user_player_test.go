package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// レート制限はパッケージ変数として共有されるため、開始時に状態をリセットした上で、
// テストごとに異なるuidを使う。

func TestUserPlayerValidation(t *testing.T) {
	userPlayerAttemptLimiterByUID.Reset()

	t.Run("UserPlayerCreateMiddleware", func(t *testing.T) {
		t.Run("正常系_player_idを受理してコンテキストに設定する", func(t *testing.T) {
			expected := dto.UserPlayerCreateRequest{PlayerId: "3000000000000001"}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-ok-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetUserPlayerCreateRequest(ctx))
			require.Equal(t, "3000000000000001", helper.GetPlayerId(ctx))
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")
			helper.SetUID(ctx, "create-badjson-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_player_idが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-empty-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_player_idが16桁を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: strings.Repeat("1", 17)})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-long-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		// 同じ player_id を別ユーザーが登録することは許容する
		t.Run("正常系_他ユーザーと同じplayer_idでも受理する", func(t *testing.T) {
			shared := "5000000000000001"

			for _, uid := range []string{"create-dup-user-a", "create-dup-user-b"} {
				b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: shared})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))
				helper.SetUID(ctx, uid)

				UserPlayerCreateMiddleware()(ctx)

				require.Equal(t, http.StatusOK, w.Code)
			}
		})

		t.Run("異常系_同一ユーザの試行回数が上限を超えたら429を返す", func(t *testing.T) {
			uid := "create-ratelimit-user"

			for i := 0; i < 10; i++ {
				b, err := json.Marshal(dto.UserPlayerCreateRequest{
					PlayerId: fmt.Sprintf("40000000000000%02d", i),
				})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))
				helper.SetUID(ctx, uid)

				UserPlayerCreateMiddleware()(ctx)

				require.Equal(t, http.StatusOK, w.Code)
			}

			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: "4000000000000099"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, uid)

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusTooManyRequests, w.Code)
		})
	})
}
