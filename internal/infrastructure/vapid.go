package infrastructure

import (
	"crypto/ecdh"
	"encoding/base64"
	"strings"
)

// VAPID 設定の検査。
//
// この設定は「間違っていても静かに壊れる」という性質を持つ。送出はプッシュサービスが
// 受理するかどうかでしか失敗が見えず、しかも FCM のレガシーエンドポイントは VAPID の
// 署名も sub も検証せずに 201 を返すのに対し、Apple は厳密に検証して 403 を返す。
// そのため設定を1つ間違えると「Android は成功しているのに iOS だけ一通も届かない」
// という、ログからは原因の分からない壊れ方になる。
//
// 2026-09 に実際に起きた: .env に RFC 8292 どおり "mailto:foo@example.com" と書いたことで
// JWT の sub が "mailto:mailto:..." になり、iOS への配信が全滅していた。
// 手で cmd/verify-vapid-keys を叩くまで誰も気づけなかった。
//
// そこで検査そのものを1箇所に集め、送出器の生成時にも必ず通す(NewWebPushSender)。
// 起動のたびにログへ出るので、次に同じ壊し方をしたら初回の実行で目に入る。

// VAPIDConfigReport は VAPID 設定の検査結果。
// Problems が空なら矛盾なし。表示用の値も持たせて、検査と出力を二重に実装しないようにする。
type VAPIDConfigReport struct {
	// Subject は設定されている値そのもの。
	Subject string
	// JWTSubject は実際に JWT の sub へ載る値(webpush-go の前置を再現したもの)。
	JWTSubject string
	// PublicKey は設定されている公開鍵(base64url の padding を除いたもの)。
	PublicKey string
	// DerivedPublicKey は秘密鍵から導出した公開鍵。秘密鍵が読めなければ空。
	DerivedPublicKey string
	// Problems は見つかった矛盾。空なら OK。
	Problems []string
}

// HasProblems は矛盾が見つかったかを返す。
func (r VAPIDConfigReport) HasProblems() bool {
	return len(r.Problems) > 0
}

// InspectVAPIDConfig は VAPID の設定を送信せずに検査する。
//
// 見るのは「設定の字面」ではなく「実際に JWT へ載る値」。webpush-go は https: で始まらない
// subject に無条件で mailto: を前置するため、設定をそのまま読んでも二重の mailto: に
// 気づけない(WebPushSender は NormalizeVAPIDSubject でこれを吸収する)。
//
// 鍵が両方とも未設定の場合は「push を使わない環境」として扱い、矛盾とはしない。
// 鍵無しでもバッチと API が起動できる設計に合わせる(Enabled() が false になるだけ)。
func InspectVAPIDConfig(publicKey string, privateKey string, subject string) VAPIDConfigReport {
	publicKey = strings.TrimSpace(publicKey)
	privateKey = strings.TrimSpace(privateKey)
	subject = strings.TrimSpace(subject)

	report := VAPIDConfigReport{
		Subject:    subject,
		JWTSubject: jwtSubject(subject),
		PublicKey:  strings.TrimRight(publicKey, "="),
	}

	if publicKey == "" && privateKey == "" {
		// push を使わない環境。subject だけ設定されていても咎めない
		return report
	}

	report.Problems = append(report.Problems, subjectProblems(subject)...)
	report.Problems = append(report.Problems, keyProblems(publicKey, privateKey, &report)...)

	return report
}

// jwtSubject は webpush-go が JWT の sub に載せる値を再現する。
// 送出器は NormalizeVAPIDSubject を通してから渡すので、同じ順序で組み立てる。
func jwtSubject(subject string) string {
	normalized := NormalizeVAPIDSubject(subject)
	if strings.HasPrefix(normalized, "https:") {
		return normalized
	}

	return "mailto:" + normalized
}

func subjectProblems(subject string) []string {
	normalized := NormalizeVAPIDSubject(subject)
	sub := jwtSubject(subject)

	switch {
	case subject == "":
		return []string{"VAPID_SUBJECT が未設定。RFC 8292 で必須なので、Apple は JWT を拒否する"}
	case normalized == "":
		return []string{"VAPID_SUBJECT が mailto: だけで連絡先が無い"}
	case strings.HasPrefix(sub, "mailto:mailto:"):
		// NormalizeVAPIDSubject があるので通常は起きないが、正規化を外したときに
		// 気づけるよう検査自体は残す(この壊れ方で iOS が全滅した実績がある)
		return []string{"JWT の sub が二重の mailto: になっている(Apple は 403 BadJwtToken を返す)"}
	case strings.HasPrefix(sub, "https:"), strings.Contains(normalized, "@"):
		return nil
	default:
		return []string{"VAPID_SUBJECT はメールアドレスか https:// の URL である必要がある"}
	}
}

func keyProblems(publicKey string, privateKey string, report *VAPIDConfigReport) []string {
	if publicKey == "" || privateKey == "" {
		return []string{"VAPID_PUBLIC_KEY と VAPID_PRIVATE_KEY は両方を設定する(片方だけでは送出できない)"}
	}

	privBytes, err := decodeVAPIDKey(privateKey)
	if err != nil {
		return []string{"VAPID_PRIVATE_KEY が base64url として読めない"}
	}

	priv, err := ecdh.P256().NewPrivateKey(privBytes)
	if err != nil {
		return []string{"VAPID_PRIVATE_KEY が P-256 の秘密鍵として不正"}
	}

	report.DerivedPublicKey = base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes())
	if report.DerivedPublicKey == report.PublicKey {
		return nil
	}

	// ここが食い違うと、購読は作れるのに署名検証だけが落ちる。
	// FCM は検証しないので 201 を返し、Apple だけが 403 を返す。
	return []string{"VAPID_PUBLIC_KEY と VAPID_PRIVATE_KEY が別の鍵ペア(webapp の NEXT_PUBLIC_VAPID_PUBLIC_KEY も揃える必要がある)"}
}

// decodeVAPIDKey は base64url(padding の有無を問わない)で符号化された鍵を復号する。
func decodeVAPIDKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}

	return base64.URLEncoding.DecodeString(s)
}
