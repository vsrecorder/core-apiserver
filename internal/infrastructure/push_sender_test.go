package infrastructure

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// newTestSubscription はペイロードの暗号化が通る形式の鍵を持つ購読を作る。
// p256dh は P-256 の公開鍵(非圧縮・65バイト)、auth は16バイトの乱数を base64url にしたもの。
func newTestSubscription(t *testing.T, endpoint string) *entity.PushSubscription {
	t.Helper()

	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)

	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)

	return entity.NewPushSubscription(
		"sub-1",
		time.Now(),
		"user-1",
		endpoint,
		base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(auth),
		entity.PushPlatformDesktop,
	)
}

func TestWebPushSender(t *testing.T) {
	t.Run("正常系_鍵が未設定ならEnabledがfalseでSendは送らずにエラーを返す", func(t *testing.T) {
		s := NewWebPushSender("", "", "")

		require.False(t, s.Enabled())

		status, err := s.Send(context.Background(), newTestSubscription(t, "https://example.com/push"), &entity.PushPayload{})

		require.ErrorIs(t, err, errPushSenderDisabled)
		require.Equal(t, 0, status)
	})

	t.Run("正常系_暗号化したペイロードをendpointへPOSTしステータスを返す", func(t *testing.T) {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		require.NoError(t, err)

		var received *http.Request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = r.Clone(context.Background())
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		s := NewWebPushSender(publicKey, privateKey, "mailto:test@example.com")
		require.True(t, s.Enabled())

		status, err := s.Send(context.Background(), newTestSubscription(t, server.URL+"/push"), &entity.PushPayload{
			Title:      "今週末、対戦の予定は？",
			Body:       "本文",
			URL:        "/records/quick",
			DeliveryId: "d-1",
			Tag:        "weekend_reminder",
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)
		require.NotNil(t, received)
		require.Equal(t, http.MethodPost, received.Method)
		require.Equal(t, "aes128gcm", received.Header.Get("Content-Encoding"))
		require.Equal(t, "86400", received.Header.Get("TTL"))
		require.Contains(t, received.Header.Get("Authorization"), "vapid")
	})

	t.Run("正常系_4xxはエラーではなくステータスとして返す", func(t *testing.T) {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusGone)
		}))
		defer server.Close()

		s := NewWebPushSender(publicKey, privateKey, "mailto:test@example.com")

		status, err := s.Send(context.Background(), newTestSubscription(t, server.URL+"/push"), &entity.PushPayload{Title: "t"})

		require.NoError(t, err)
		require.Equal(t, http.StatusGone, status)
	})
}
