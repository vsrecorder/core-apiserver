package infrastructure

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

	// 拒否の理由はプッシュサービスが本文で返す。これを残さないと、403 が鍵・subject・JWT の
	// どれによるものか切り分けられない(iOS へ一通も届かない事故の調査で実際に行き詰まった)
	t.Run("正常系_拒否されたらレスポンス本文を理由としてログに残す", func(t *testing.T) {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"reason":"BadJwtToken"}`))
		}))
		defer server.Close()

		var logs bytes.Buffer
		previous := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
		t.Cleanup(func() { slog.SetDefault(previous) })

		s := NewWebPushSender(publicKey, privateKey, "mailto:test@example.com")

		status, err := s.Send(context.Background(), newTestSubscription(t, server.URL+"/push"), &entity.PushPayload{Title: "t"})

		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, status)
		require.Contains(t, logs.String(), "BadJwtToken")
		// どのプッシュサービスが拒否したのかも分かるようにする
		require.Contains(t, logs.String(), "push_service")
	})

	// webpush-go は "https:" で始まらない subject に "mailto:" を前置する。
	// .env に RFC 8292 どおり "mailto:..." と書くと sub が "mailto:mailto:..." になり、
	// Apple だけが 403 BadJwtToken で弾く(FCM は sub を見ないので 201 を返す)
	t.Run("正常系_subjectのmailto:が二重にならない", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			subject string
			want    string
		}{
			{"mailto:付き", "mailto:contact@example.com", "mailto:contact@example.com"},
			{"mailto:無し", "contact@example.com", "mailto:contact@example.com"},
			{"httpsのURL", "https://example.com/contact", "https://example.com/contact"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
				require.NoError(t, err)

				var authorization string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					authorization = r.Header.Get("Authorization")
					w.WriteHeader(http.StatusCreated)
				}))
				defer server.Close()

				s := NewWebPushSender(publicKey, privateKey, tc.subject)

				_, err = s.Send(context.Background(), newTestSubscription(t, server.URL+"/push"), &entity.PushPayload{Title: "t"})
				require.NoError(t, err)

				require.Equal(t, tc.want, vapidSubjectOf(t, authorization))
			})
		}
	})
}

// vapidSubjectOf は Authorization ヘッダ("vapid t=<JWT>, k=<公開鍵>")から JWT を取り出し、
// sub クレームを返す。プッシュサービスが実際に検証するのはこの値。
func vapidSubjectOf(t *testing.T, authorization string) string {
	t.Helper()

	require.NotEmpty(t, authorization)

	var token string
	for _, part := range strings.Split(strings.TrimPrefix(authorization, "vapid "), ",") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "t="); ok {
			token = after
		}
	}
	require.NotEmpty(t, token, "Authorization に t=<JWT> が無い: %s", authorization)

	segments := strings.Split(token, ".")
	require.Len(t, segments, 3, "JWT の形式が不正: %s", token)

	payload, err := base64.RawURLEncoding.DecodeString(segments[1])
	require.NoError(t, err)

	var claims struct {
		Sub string `json:"sub"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))

	return claims.Sub
}
