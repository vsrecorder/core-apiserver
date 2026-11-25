// Package batchreport は、利用者ごとに処理する定期バッチの結果の締め方をそろえる。
//
// 通知系のバッチ(週次レポート・連続記録の声かけ・週末のリマインド・期間レポート)は、利用者 1 人の
// 失敗で全体を止めないよう、失敗をログに出して次の利用者へ進む。以前はそのまま正常終了していたので、
// DB が落ちたなどで全員ぶん失敗しても終了コードは 0 で、誰にも通知が届いていないことに気づく手段が
// ログを読むことしか無かった(push の点検 check-push-health は、配達が 3 件に満たない区分を
// 判定しないので、「1 通も送っていない」は拾えない)。
//
// 失敗が 1 人でもあれば、まとめを Error で出し、指定されていれば Slack へ知らせ、非 0 で終える(Finish)。
//
// データベースに繋がらない・対象の一覧を取れないなど、利用者ごとの処理に入る前にバッチ全体が
// 止まった場合も、同じ先へ知らせて非 0 で終える(Abort)。こちらは以前から非 0 で終えていたが、
// 知らせてはいなかった。DB が落ちているときはまずここで止まるので、知らせないと Finish の通知まで届かない。
package batchreport

import (
	"fmt"
	"log/slog"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// Result はバッチ 1 回ぶんの結果。
type Result struct {
	// AppName はバッチの名前(cmd のディレクトリ名)。
	AppName string
	// Candidates は対象にした利用者の数。
	Candidates int
	// Failed は処理に失敗した利用者の数。
	Failed int
	// ListFailed は、対象の利用者の一覧そのものを取れなかった回数(期間レポートのように対象が複数ある場合)。
	ListFailed int
}

// HasFailure は、知らせるべき失敗があるかを返す。
func (r Result) HasFailure() bool {
	return r.Failed > 0 || r.ListFailed > 0
}

// Message は Slack に出す文面を返す。
func (r Result) Message() string {
	message := fmt.Sprintf(":warning: バッチ %s で失敗がありました\n", r.AppName)

	if r.Failed > 0 {
		message += fmt.Sprintf("• 対象 %d 人のうち %d 人の処理に失敗", r.Candidates, r.Failed)
		if r.Failed == r.Candidates {
			message += "(全員)"
		}
		message += "\n"
	}
	if r.ListFailed > 0 {
		message += fmt.Sprintf("• 対象の利用者の一覧を %d 回取得できませんでした\n", r.ListFailed)
	}

	message += "詳しくはサーバのバッチのログ(/var/log/vsrecorder/)を確認してください"

	return message
}

// Notifier は失敗を外へ知らせる(Slack の incoming webhook への投稿)。
type Notifier func(message string) error

// Finish は結果を締め、プロセスの終了コードを返す。
//
// 失敗が無ければ何もせず OK。失敗があればまとめを Error で出し、notify が渡されていれば知らせて NG。
// 知らせること自体に失敗しても NG のまま終える(ログには残っている)。
func Finish(r Result, notify Notifier) int {
	if !r.HasFailure() {
		return ExitCodeOK
	}

	slog.Error("batch finished with failures",
		slog.Int("candidate_users", r.Candidates),
		slog.Int("failed_users", r.Failed),
		slog.Int("failed_lists", r.ListFailed),
	)

	if notify != nil {
		if err := notify(r.Message()); err != nil {
			slog.Error("failed to notify the failures to slack", slog.String("error_message", err.Error()))
		} else {
			slog.Info("notified the failures to slack")
		}
	}

	return ExitCodeNG
}

// AbortMessage は、バッチ全体が続けられなくなったときに Slack に出す文面を返す。
func AbortMessage(appName string, reason string) string {
	return fmt.Sprintf(":warning: バッチ %s が途中で止まりました\n• %s\n", appName, reason) +
		"詳しくはサーバのバッチのログ(/var/log/vsrecorder/)を確認してください"
}

// Abort は、バッチ全体が続けられなくなったときに締め、プロセスの終了コード(常に NG)を返す。
//
// reason は止まった場面(「データベースに接続できませんでした」など)。エラーの中身は接続先などを
// 含みうるので Slack へは出さない。呼び出し側が先にログへ出しておくこと。
// 知らせること自体に失敗しても NG のまま終える。
func Abort(appName string, reason string, notify Notifier) int {
	if notify != nil {
		if err := notify(AbortMessage(appName, reason)); err != nil {
			slog.Error("failed to notify the abort to slack", slog.String("error_message", err.Error()))
		} else {
			slog.Info("notified the abort to slack")
		}
	}

	return ExitCodeNG
}
