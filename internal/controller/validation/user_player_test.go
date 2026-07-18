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

// レート制限はパッケージ変数として共有されるため、テスト間の干渉を避けるべく
// テストごとに異なるuid・player_idを使う。

func TestUserPlayerValidation(t *testing.T) {
	t.Run("UserPlayerVerifyMiddleware", func(t *testing.T) {
		t.Run("正常系_player_idを受理してコンテキストに設定する", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerVerifyRequest{PlayerId: "1000000000000001"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "verify-ok-user")

			UserPlayerVerifyMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "1000000000000001", helper.GetPlayerId(ctx))
			require.Equal(t, "1000000000000001", helper.GetUserPlayerVerifyRequest(ctx).PlayerId)
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")
			helper.SetUID(ctx, "verify-badjson-user")

			UserPlayerVerifyMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_player_idが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerVerifyRequest{PlayerId: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "verify-empty-user")

			UserPlayerVerifyMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_player_idが16桁を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerVerifyRequest{PlayerId: strings.Repeat("1", 17)})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "verify-long-user")

			UserPlayerVerifyMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		// 総当たり防止のレート制限(uid単位で1時間に10回)を超えると429になる
		t.Run("異常系_同一ユーザの試行回数が上限を超えたら429を返す", func(t *testing.T) {
			uid := "verify-ratelimit-user"

			for i := 0; i < 10; i++ {
				b, err := json.Marshal(dto.UserPlayerVerifyRequest{PlayerId: fmt.Sprintf("20000000000000%02d", i)})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))
				helper.SetUID(ctx, uid)

				UserPlayerVerifyMiddleware()(ctx)

				require.Equal(t, http.StatusOK, w.Code)
			}

			b, err := json.Marshal(dto.UserPlayerVerifyRequest{PlayerId: "2000000000000099"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, uid)

			UserPlayerVerifyMiddleware()(ctx)

			require.Equal(t, http.StatusTooManyRequests, w.Code)
		})
	})

	t.Run("UserPlayerCreateMiddleware", func(t *testing.T) {
		t.Run("正常系_player_idとチャレンジトークンを受理して設定する", func(t *testing.T) {
			expected := dto.UserPlayerCreateRequest{
				PlayerId:       "3000000000000001",
				ChallengeToken: "token",
			}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-ok-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetUserPlayerCreateRequest(ctx))
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")
			helper.SetUID(ctx, "create-badjson-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_player_idが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: "", ChallengeToken: "token"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-empty-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_チャレンジトークンが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: "3000000000000002", ChallengeToken: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-notoken-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
