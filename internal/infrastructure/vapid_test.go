package infrastructure

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

// newVAPIDKeyPair は検査用の P-256 鍵ペアを base64url で返す。
func newVAPIDKeyPair(t *testing.T) (publicKey string, privateKey string) {
	t.Helper()

	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)

	return base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(priv.Bytes())
}

func TestInspectVAPIDConfig(t *testing.T) {
	pub, priv := newVAPIDKeyPair(t)

	t.Run("正常系_鍵ペアが一致しsubjectがメールアドレスなら矛盾なし", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, priv, "mailto:contact@example.com")

		require.False(t, report.HasProblems(), "problems: %v", report.Problems)
		require.Equal(t, "mailto:contact@example.com", report.JWTSubject)
		require.Equal(t, pub, report.DerivedPublicKey)
	})

	t.Run("正常系_httpsのsubjectも妥当", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, priv, "https://vsrecorder.mobi/contact")

		require.False(t, report.HasProblems(), "problems: %v", report.Problems)
		// https: は前置されない
		require.Equal(t, "https://vsrecorder.mobi/contact", report.JWTSubject)
	})

	/*
	 * 2026-09 に iOS への配信を全滅させた設定。
	 * .env に RFC 8292 どおり mailto: を書くと webpush-go がもう一度前置するため
	 * sub が "mailto:mailto:..." になり、Apple だけが 403 BadJwtToken で弾く。
	 * いまは NormalizeVAPIDSubject が剥がすので通常は起きないが、正規化を外したり
	 * 別のライブラリへ差し替えたときに黙って再発しないよう、検査自体を残しておく。
	 */
	t.Run("二重のmailtoを検出する", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, priv, "mailto:mailto:contact@example.com")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "二重の mailto:")
		require.Equal(t, "mailto:mailto:contact@example.com", report.JWTSubject)
	})

	t.Run("鍵ペアが食い違っていたら検出する", func(t *testing.T) {
		// 公開鍵だけ別の鍵ペアのものにする。購読は作れるのに署名検証だけが落ちる状態
		otherPub, _ := newVAPIDKeyPair(t)

		report := InspectVAPIDConfig(otherPub, priv, "mailto:contact@example.com")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "別の鍵ペア")
	})

	t.Run("subjectが未設定なら検出する", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, priv, "")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "未設定")
	})

	t.Run("subjectがメールでもURLでもないなら検出する", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, priv, "vsrecorder")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "メールアドレスか https://")
	})

	t.Run("mailtoだけで連絡先が無いなら検出する", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, priv, "mailto:")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "連絡先が無い")
	})

	t.Run("秘密鍵が読めないなら検出する", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, "not-a-key!!", "mailto:contact@example.com")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "base64url")
	})

	t.Run("片方の鍵だけ設定されていたら検出する", func(t *testing.T) {
		report := InspectVAPIDConfig(pub, "", "mailto:contact@example.com")

		require.True(t, report.HasProblems())
		require.Contains(t, report.Problems[0], "両方を設定する")
	})

	// 鍵が無い環境でもバッチとAPIは起動する。push を使わないだけなので咎めない
	t.Run("鍵が両方とも未設定なら矛盾としない", func(t *testing.T) {
		report := InspectVAPIDConfig("", "", "")

		require.False(t, report.HasProblems(), "problems: %v", report.Problems)
	})

	// padding 付きで設定されていても同じ鍵として扱う
	t.Run("公開鍵のpaddingは判定に影響しない", func(t *testing.T) {
		report := InspectVAPIDConfig(pub+"==", priv, "mailto:contact@example.com")

		require.False(t, report.HasProblems(), "problems: %v", report.Problems)
	})
}
