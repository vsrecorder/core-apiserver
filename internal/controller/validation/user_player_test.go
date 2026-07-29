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
// テストごとに異なるuid・player_idを使う。

func TestUserPlayerValidation(t *testing.T) {
	userPlayerAttemptLimiterByUID.Reset()
	userPlayerAttemptLimiterByPlayerID.Reset()

	t.Run("UserPlayerChallengeMiddleware", func(t *testing.T) {
		t.Run("正常系_current_avatar_imageを受理してコンテキストに設定する", func(t *testing.T) {
			expected := dto.UserPlayerChallengeRequest{
				CurrentAvatarImage: "https://example.com/current-avatar.png",
			}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "challenge-ok-user")

			UserPlayerChallengeMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetUserPlayerChallengeRequest(ctx))
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")
			helper.SetUID(ctx, "challenge-badjson-user")

			UserPlayerChallengeMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		// 除外条件が空でも「どのアバターでもよい」という意味になるため受理する
		t.Run("正常系_current_avatar_imageが空でも受理する", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerChallengeRequest{CurrentAvatarImage: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "challenge-empty-user")

			UserPlayerChallengeMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("UserPlayerCreateMiddleware", func(t *testing.T) {
		t.Run("正常系_player_idと検証済みトークンを受理して設定する", func(t *testing.T) {
			expected := dto.UserPlayerCreateRequest{
				PlayerId:          "3000000000000001",
				VerificationToken: "token",
			}

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
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: "", VerificationToken: "token"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-empty-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_player_idが16桁を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: strings.Repeat("1", 17), VerificationToken: "token"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-long-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_検証済みトークンが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: "3000000000000002", VerificationToken: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, "create-notoken-user")

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		// 総当たり防止のレート制限(uid単位で1時間に10回)を超えると429になる
		t.Run("異常系_同一ユーザの試行回数が上限を超えたら429を返す", func(t *testing.T) {
			uid := "create-ratelimit-user"

			for i := 0; i < 10; i++ {
				b, err := json.Marshal(dto.UserPlayerCreateRequest{
					PlayerId:          fmt.Sprintf("40000000000000%02d", i),
					VerificationToken: "token",
				})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))
				helper.SetUID(ctx, uid)

				UserPlayerCreateMiddleware()(ctx)

				require.Equal(t, http.StatusOK, w.Code)
			}

			b, err := json.Marshal(dto.UserPlayerCreateRequest{
				PlayerId:          "4000000000000099",
				VerificationToken: "token",
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))
			helper.SetUID(ctx, uid)

			UserPlayerCreateMiddleware()(ctx)

			require.Equal(t, http.StatusTooManyRequests, w.Code)
		})
	})
}
