package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeckCardFindAceSpec(t *testing.T) {
	t.Run("正常系_card_idが数値で返っても文字列として受ける", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/v1beta/deckcards/cYGaDx-IOYQzZ-8cJYx8/acespec", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"card_id":46197,"card_name":"パーフェクトミキサー","detail_url":"https://example.com","image_url":"https://example.com/a.jpg"}`))
		}))
		defer server.Close()

		card, err := NewDeckCard(server.URL).FindAceSpec(context.Background(), "cYGaDx-IOYQzZ-8cJYx8")

		require.NoError(t, err)
		require.NotNil(t, card)
		require.Equal(t, "46197", card.CardId)
		require.Equal(t, "パーフェクトミキサー", card.CardName)
		require.Equal(t, "https://example.com/a.jpg", card.ImageURL)
	})

	t.Run("正常系_204は該当なし", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		card, err := NewDeckCard(server.URL).FindAceSpec(context.Background(), "kVkFF5-pQ2sZa-VFVfkV")

		require.NoError(t, err)
		require.Nil(t, card)
	})

	t.Run("異常系_5xxはエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := NewDeckCard(server.URL).FindAceSpec(context.Background(), "kVkFF5-pQ2sZa-VFVfkV")

		require.Error(t, err)
	})
}
