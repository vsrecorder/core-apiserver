package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotify(t *testing.T) {
	t.Run("正常系_JSONのtextとして送信される", func(t *testing.T) {
		var (
			gotContentType string
			gotBody        []byte
		)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotContentType = r.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		err := Notify(server.URL, "差異があります")

		assert.NoError(t, err)
		assert.Equal(t, "application/json", gotContentType)

		var payload struct {
			Text string `json:"text"`
		}
		assert.NoError(t, json.Unmarshal(gotBody, &payload))
		assert.Equal(t, "差異があります", payload.Text)
	})

	t.Run("異常系_200以外はボディを添えてエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid_payload"))
		}))
		defer server.Close()

		err := Notify(server.URL, "差異があります")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "invalid_payload")
	})

	t.Run("異常系_送信先へ到達できなければエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close()

		err := Notify(url, "差異があります")

		assert.Error(t, err)
	})

	// 通知するつもりで送信先が無い状態は、黙って成功扱いにすると気付けない
	t.Run("異常系_webhookURLが空ならエラーを返す", func(t *testing.T) {
		err := Notify("", "差異があります")

		require.ErrorIs(t, err, ErrWebhookURLNotSet)
	})
}
