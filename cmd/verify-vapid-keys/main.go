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
 * 判定そのものは infrastructure.InspectVAPIDConfig が持つ。送出器(NewWebPushSender)も
 * 同じ関数を通すので、「手で叩けば分かるのに本番では素通りする」という食い違いが起きない。
 * ここが受け持つのは表示だけ。
 *
 * 秘密鍵そのものは決して出力しない。
 */
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	_ = godotenv.Load()

	report := infrastructure.InspectVAPIDConfig(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_SUBJECT"),
	)

	// 見るべきは .env の字面ではなく、実際に JWT の sub へ載る値。
	// webpush-go は "https:" で始まらない subject に "mailto:" を前置するため、
	// 設定をそのまま渡すと "mailto:mailto:..." になりうる(Apple は 403 BadJwtToken)。
	fmt.Printf("VAPID_SUBJECT   : %q\n", report.Subject)
	fmt.Printf("JWT の sub      : %q\n", report.JWTSubject)

	fmt.Println()
	fmt.Printf("VAPID_PUBLIC_KEY : %s\n", report.PublicKey)
	if report.DerivedPublicKey != "" {
		fmt.Printf("秘密鍵から導出   : %s\n", report.DerivedPublicKey)
	}

	if !report.HasProblems() {
		fmt.Println("\nVAPID の設定に矛盾は見つからなかった")
		os.Exit(ExitCodeOK)
	}

	fmt.Println("\n見つかった矛盾:")
	for _, problem := range report.Problems {
		fmt.Printf("  NG: %s\n", problem)
	}
	fmt.Println()
	fmt.Println("  FCM(レガシーendpoint)は署名も sub も検証しないため 201 を返すが、")
	fmt.Println("  Apple は 403 を返し iOS には一通も届かない。")
	fmt.Println("  鍵を揃え直した場合は webapp の NEXT_PUBLIC_VAPID_PUBLIC_KEY も合わせ、")
	fmt.Println("  webapp を再ビルドすること(既存の購読は作り直しになる)。")

	os.Exit(ExitCodeNG)
}
