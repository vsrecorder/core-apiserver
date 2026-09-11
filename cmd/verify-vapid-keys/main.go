/*
 * VAPID 設定の健全性を確認する。
 *
 * Web Push の送出は、鍵や subject が間違っていても「プッシュサービスが受け取るかどうか」
 * でしか失敗が見えない。しかも FCM のレガシーエンドポイント(fcm.googleapis.com/fcm/send/)は
 * VAPID の署名を検証せずに 201 を返すため、設定が壊れていても Android では成功したように
 * 見える。一方 Apple(web.push.apple.com)は厳密に検証して 403 を返す。
 * 結果として「iOS だけ一通も届かないが、原因がログから分からない」状態になる。
 *
 * このコマンドは送信を伴わずに、設定そのものの矛盾を洗い出す:
 *   - 公開鍵・秘密鍵が同じ鍵ペアか(秘密鍵から公開鍵を導出して突き合わせる)
 *   - subject が RFC 8292 の要求する mailto: / https: URL か
 *
 * 秘密鍵そのものは決して出力しない。
 */
package main

import (
	"crypto/ecdh"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// decodeKey は base64url(padding の有無を問わない)で符号化された鍵を復号する。
func decodeKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func main() {
	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	_ = godotenv.Load()

	publicKey := strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY"))
	privateKey := strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY"))
	subject := strings.TrimSpace(os.Getenv("VAPID_SUBJECT"))

	ok := true

	// --- subject ---
	// RFC 8292 §2.1: sub は連絡先を表す mailto: か https: の URI。
	// Apple はこれを満たさない JWT を 403 で拒否する。
	fmt.Printf("VAPID_SUBJECT   : %q\n", subject)
	switch {
	case subject == "":
		fmt.Println("  NG: 未設定。web push そのものが無効になる")
		ok = false
	case strings.HasPrefix(subject, "mailto:"), strings.HasPrefix(subject, "https://"):
		fmt.Println("  OK: mailto: / https: で始まっている")
	default:
		fmt.Println("  NG: mailto: か https:// で始まる必要がある(Apple は 403 を返す)")
		ok = false
	}

	// --- 鍵ペア ---
	fmt.Println()
	if publicKey == "" || privateKey == "" {
		fmt.Println("VAPID の鍵が未設定(VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY)")
		os.Exit(ExitCodeNG)
	}

	privBytes, err := decodeKey(privateKey)
	if err != nil {
		fmt.Printf("VAPID_PRIVATE_KEY: NG: base64url として読めない: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	priv, err := ecdh.P256().NewPrivateKey(privBytes)
	if err != nil {
		fmt.Printf("VAPID_PRIVATE_KEY: NG: P-256 の秘密鍵として不正(%d バイト): %v\n", len(privBytes), err)
		os.Exit(ExitCodeNG)
	}

	// 秘密鍵から導出した公開鍵と、設定されている公開鍵が一致するかを見る。
	// ここが食い違っていると、購読は作れるのに署名検証だけが落ちる。
	derived := base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes())
	configured := strings.TrimRight(publicKey, "=")

	fmt.Printf("VAPID_PUBLIC_KEY : %s\n", configured)
	fmt.Printf("秘密鍵から導出   : %s\n", derived)

	if derived == configured {
		fmt.Println("  OK: 公開鍵と秘密鍵は同じ鍵ペア")
	} else {
		fmt.Println("  NG: 公開鍵と秘密鍵が別の鍵ペア。")
		fmt.Println("      FCM(レガシーendpoint)は署名を検証しないため 201 を返すが、")
		fmt.Println("      Apple は 403 を返し iOS には一通も届かない。")
		fmt.Println("      webapp の NEXT_PUBLIC_VAPID_PUBLIC_KEY もこの公開鍵に揃えること")
		fmt.Println("      (揃え直したら webapp の再ビルドが要る。既存の購読は作り直しになる)")
		ok = false
	}

	if !ok {
		os.Exit(ExitCodeNG)
	}

	fmt.Println("\nVAPID の設定に矛盾は見つからなかった")
	os.Exit(ExitCodeOK)
}
