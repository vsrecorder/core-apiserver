package usecase

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

// overridePlayerAccountAPI はプレイヤーズクラブAPIの取得先をhttptestサーバへ差し替える
// (外部サイトへは通信しない)。
func overridePlayerAccountAPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := playerAccountAPIURL
	playerAccountAPIURL = server.URL
	t.Cleanup(func() { playerAccountAPIURL = original })
}

func TestFetchPlayerAccount(t *testing.T) {
	playerId := "1234567890123456"

	t.Run("正常系_実在するプレイヤーの情報を返す", func(t *testing.T) {
		overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
			require.NoError(t, req.ParseForm())
			require.Equal(t, playerId, req.PostForm.Get("player_id"))

			fmt.Fprint(w, `{"code":200,"player":{
				"player_id":"1234567890123456",
				"nickname":"ニックネーム",
				"avatar_image":"https://example.com/avatar.png",
				"current_league":"マスター",
				"prefecture":"福岡県"
			}}`)
		})

		ret, err := fetchPlayerAccount(playerId)

		require.NoError(t, err)
		require.Equal(t, playerId, ret.PlayerId)
		require.Equal(t, "ニックネーム", ret.Nickname)
		require.Equal(t, "https://example.com/avatar.png", ret.AvatarImage)
		require.Equal(t, "マスター", ret.CurrentLeague)
		require.Equal(t, "福岡県", ret.Prefecture)
	})

	t.Run("異常系_存在しないプレイヤーはErrRecordNotFoundを返す", func(t *testing.T) {
		// このAPIは存在しない場合も200以外のステータス+JSONボディで返すため、
		// ボディのcodeで判定される
		overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"code":404,"message":"not found"}`)
		})

		_, err := fetchPlayerAccount(playerId)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})

	t.Run("異常系_codeが200でもplayerが無ければErrRecordNotFoundを返す", func(t *testing.T) {
		overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, `{"code":200}`)
		})

		_, err := fetchPlayerAccount(playerId)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})

	t.Run("異常系_JSONとして不正なレスポンスはエラーを返す", func(t *testing.T) {
		overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, `not json`)
		})

		_, err := fetchPlayerAccount(playerId)

		require.Error(t, err)
	})
}
