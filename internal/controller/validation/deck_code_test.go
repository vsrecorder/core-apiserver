package validation

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestDeckCodeValidation(t *testing.T) {
	// overrideDeckIDCheckAPI はデッキコードの実在確認APIをhttptestサーバへ差し替える
	// (公式サイトへは通信しない)。
	overrideDeckIDCheckAPI := func(t *testing.T, handler http.HandlerFunc) {
		t.Helper()

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		original := DeckIDCheckURL
		DeckIDCheckURL = server.URL
		t.Cleanup(func() { DeckIDCheckURL = original })
	}

	t.Run("DeckCodeCreateMiddleware", func(t *testing.T) {
		t.Run("正常系_実在するデッキコードのリクエストを受理して設定する", func(t *testing.T) {
			overrideDeckIDCheckAPI(t, func(w http.ResponseWriter, req *http.Request) {
				require.NoError(t, req.ParseForm())
				require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", req.PostForm.Get("deckID"))
				fmt.Fprint(w, `{"result":1,"deckID":"5dbFbk-uBwjqP-VVk5Vv","existence":1}`)
			})

			expected := dto.DeckCodeCreateRequest{
				DeckId: "01HD7Y3K8D6FDHMHTZ2GT41TD1",
				Code:   "5dbFbk-uBwjqP-VVk5Vv",
			}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetDeckCodeCreateRequest(ctx))
		})

		// フロントエンド側でデッキコードの有効性を確認しているのでチェックは不要だが、念のためサーバ側でも確認したいが、
		// 大量のリクエストが来ると外部APIに負荷がかかるので、現状はコメントアウトしている。
		// もし外部APIの負荷が問題ない場合は、コメントアウトを解除してチェックを有効化する。
		/*
			t.Run("異常系_実在しないデッキコードなら400を返す", func(t *testing.T) {
				overrideDeckIDCheckAPI(t, func(w http.ResponseWriter, req *http.Request) {
					fmt.Fprint(w, `{"result":0,"errMsg":"deck not found","existence":0}`)
				})

				b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: "XXXXXX-YYYYYY-ZZZZZZ"})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))

				DeckCodeCreateMiddleware(slog.Default())(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
			})
		*/

		// フロントエンド側でデッキコードの有効性を確認しているのでチェックは不要だが、念のためサーバ側でも確認したいが、
		// 大量のリクエストが来ると外部APIに負荷がかかるので、現状はコメントアウトしている。
		// もし外部APIの負荷が問題ない場合は、コメントアウトを解除してチェックを有効化する。
		/*
			t.Run("異常系_確認先がメンテナンス中なら503を返す", func(t *testing.T) {
				overrideDeckIDCheckAPI(t, func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				})

				b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: "5dbFbk-uBwjqP-VVk5Vv"})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))

				DeckCodeCreateMiddleware(slog.Default())(ctx)

				require.Equal(t, http.StatusServiceUnavailable, w.Code)
			})
		*/

		t.Run("異常系_スプライトが3体以上なら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, `{"deck_id":"01HD7Y3K8D6FDHMHTZ2GT41TD1","code":"5dbFbk-uBwjqP-VVk5Vv","pokemon_sprites":[{"id":"0887"},{"id":"0006"},{"id":"0257"}]}`)

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_コードが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_コードが上限を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: strings.Repeat("x", MaxDeckCodeLength+1)})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_メモが上限を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeCreateRequest{
				Code: "5dbFbk-uBwjqP-VVk5Vv",
				Memo: strings.Repeat("あ", MaxMemoLength+1),
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("DeckCodeUpdateMiddleware", func(t *testing.T) {
		t.Run("正常系_更新リクエストを受理してコンテキストに設定する", func(t *testing.T) {
			expected := dto.DeckCodeUpdateRequest{PrivateCodeFlg: true, Memo: "メモ"}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetDeckCodeUpdateRequest(ctx))
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		// スプライトは「項目ごと省略」と「空配列」で意味が違う(省略は現在の設定を保ち、
		// 空配列は未設定に戻す)。入口でこの区別が潰れると、メモだけを更新する古いクライアントが
		// 設定済みのスプライトを消してしまう。
		t.Run("正常系_スプライトの省略はnil_空配列は空のまま渡る", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, `{"private_code_flg":true,"memo":"メモ","tag_ids":[]}`)
			DeckCodeUpdateMiddleware()(ctx)
			require.Equal(t, http.StatusOK, w.Code)
			require.Nil(t, helper.GetDeckCodeUpdateRequest(ctx).PokemonSprites)

			ctx, w = newValidationJSONContext(t, `{"private_code_flg":true,"memo":"メモ","tag_ids":[],"pokemon_sprites":[]}`)
			DeckCodeUpdateMiddleware()(ctx)
			require.Equal(t, http.StatusOK, w.Code)
			require.NotNil(t, helper.GetDeckCodeUpdateRequest(ctx).PokemonSprites)
			require.Empty(t, helper.GetDeckCodeUpdateRequest(ctx).PokemonSprites)
		})

		t.Run("正常系_スプライトを枠つきで受理する", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, `{"pokemon_sprites":[{"id":"0887","position":1},{"id":"0006","position":2}]}`)

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			sprites := helper.GetDeckCodeUpdateRequest(ctx).PokemonSprites
			require.Len(t, sprites, 2)
			require.Equal(t, "0006", sprites[1].ID)
			require.Equal(t, uint(2), sprites[1].Position)
		})

		t.Run("異常系_スプライトが3体以上または枠が重複なら400を返す", func(t *testing.T) {
			for _, body := range []string{
				`{"pokemon_sprites":[{"id":"0887"},{"id":"0006"},{"id":"0257"}]}`,
				`{"pokemon_sprites":[{"id":"0887","position":1},{"id":"0006","position":1}]}`,
			} {
				ctx, w := newValidationJSONContext(t, body)

				DeckCodeUpdateMiddleware()(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code, body)
			}
		})

		t.Run("異常系_メモが上限を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeUpdateRequest{Memo: strings.Repeat("あ", MaxMemoLength+1)})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
