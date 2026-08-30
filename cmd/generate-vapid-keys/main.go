// generate-vapid-keys は Web Push(VAPID)の鍵ペアを1組生成して標準出力に表示するだけのツール。
// DB にも外部にも接続しない。
//
// 出力をそのまま core-apiserver の .env に貼り、公開鍵だけを webapp の
// NEXT_PUBLIC_VAPID_PUBLIC_KEY にも設定する(B1_B2_PUSH_NOTIFICATION_PLAN.md §5.6)。
//
// ■ 鍵は生成したらもう変えない。秘密鍵を変更すると、その鍵で作られた既存の購読が
//   すべて無効になり、全員に許諾を取り直すことになる(許諾の再取得は回復不能に近い)。
//   本番用の鍵はバックアップを取り、開発用とは別に生成する。
//
// 使い方:
//
//	go run ./cmd/generate-vapid-keys
package main

import (
	"fmt"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate VAPID keys: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	fmt.Println("# core-apiserver の .env に貼る(秘密鍵は webapp に置かない)")
	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", publicKey)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", privateKey)
	fmt.Println("VAPID_SUBJECT=mailto:contact@vsrecorder.mobi")
	fmt.Println()
	fmt.Println("# webapp の .env に貼る(公開鍵のみ)")
	fmt.Printf("NEXT_PUBLIC_VAPID_PUBLIC_KEY=%s\n", publicKey)

	os.Exit(ExitCodeOK)
}
