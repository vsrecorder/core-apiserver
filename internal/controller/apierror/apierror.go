// Package apierror はコントローラ層で扱う、HTTPステータスコードを内包した
// 独自エラーを定義する。
//
// ドメイン層のエラー(apperror)が「何が起きたか」を表すのに対し、apierror は
// 「クライアントへどのステータス・メッセージで応答するか」という HTTP 表現を
// 担う。各ハンドラは応答内容を文字列リテラルで直接書く代わりに、ここで定義した
// 値を JSON(ctx) で返すことで、ステータスとメッセージの対応を一元管理できる。
package apierror

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/logging"
)

// Error は HTTP ステータスコードとエラー内容を保持する独自エラー。
type Error struct {
	status int
	err    error
}

// New は指定したステータスコードとエラーから Error を生成する。
func New(status int, err error) *Error {
	return &Error{
		status: status,
		err:    err,
	}
}

// Error は error インターフェースを満たす。
func (e *Error) Error() string {
	return e.err.Error()
}

// Unwrap はラップした元エラーを返し、errors.Is / errors.As に対応する。
func (e *Error) Unwrap() error {
	return e.err
}

// Status は保持している HTTP ステータスコードを返す。
func (e *Error) Status() int {
	return e.status
}

// JSON は gin コンテキストへステータスコードとメッセージを書き込み、
// 後続ハンドラの実行を中断する。同時にコントローラ層のログも出力する。
//
// cause には応答の原因となったエラーを渡せる(可変長にしているのは、原因を
// 持たない呼び出し側をそのまま使えるようにするため)。エラー応答を返す経路は
// 必ずここを通るため、ログの書き漏れが構造上起きない。
func (e *Error) JSON(ctx *gin.Context, cause ...error) {
	e.log(ctx, cause...)

	ctx.JSON(e.status, gin.H{"message": e.err.Error()})
	ctx.Abort()
}

// log はエラー応答をコントローラ層のログとして出力する。
func (e *Error) log(ctx *gin.Context, cause ...error) {
	// 5xx はサーバ側の異常、4xx はクライアント起因の想定内の失敗であるため
	// レベルを分け、アラート対象を 5xx に絞れるようにしている。
	level := slog.LevelWarn
	if e.status >= http.StatusInternalServerError {
		level = slog.LevelError
	}

	attrs := []slog.Attr{
		// skip=2 は log → JSON → 呼び出し元(ハンドラ/ミドルウェア)。
		logging.Operation(logging.CallerOperation(2)),
		slog.Int("status_code", e.status),
		slog.String("response_message", e.err.Error()),
	}

	if len(cause) > 0 && cause[0] != nil {
		attrs = append(attrs, logging.Err(cause[0]))
	}

	logging.LogAt(
		requestContext(ctx),
		logging.Layered(logging.LayerController),
		level,
		2,
		"error response",
		attrs...,
	)
}

// requestContext は request_id / uid を載せた *http.Request の context を返す。
//
// gin.Context 自体も context.Context を満たすが、値の参照を Request の context へ
// 委譲するのは ContextWithFallback を有効にした場合だけなので、確実に値を拾える
// Request の context を直接使う。
func requestContext(ctx *gin.Context) context.Context {
	if ctx == nil || ctx.Request == nil {
		return context.Background()
	}

	return ctx.Request.Context()
}

// 定義済みエラー。コントローラ層で頻出する応答を集約する。
var (
	// ErrBadRequest はリクエストが不正な場合(400)。
	ErrBadRequest = New(http.StatusBadRequest, errors.New("bad request"))

	// ErrBadRequestNotFound は対象リソースが存在しない場合に 400 で返す既存挙動用。
	// (Delete 系ハンドラが 404 ではなく 400 を返しているため互換目的で保持している)
	ErrBadRequestNotFound = New(http.StatusBadRequest, errors.New("not found"))

	// ErrUnauthorized は認証されていない場合(401)。
	ErrUnauthorized = New(http.StatusUnauthorized, errors.New("unauthorized"))

	// ErrForbidden は権限がない場合(403)。
	ErrForbidden = New(http.StatusForbidden, errors.New("forbidden"))

	// ErrNotFound は対象リソースが存在しない場合(404)。
	ErrNotFound = New(http.StatusNotFound, errors.New("not found"))

	// ErrConflict は作成しようとしたリソースが既に存在する場合(409)。
	ErrConflict = New(http.StatusConflict, errors.New("already exists"))

	// ErrGone は対象が退会済みで、恒久的に失われている場合(410)。
	// 404 と区別することで、クライアントは「未登録なので作成してよい」のか
	// 「退会済みなので作成してはいけない」のかを判断できる。
	ErrGone = New(http.StatusGone, errors.New("withdrawn"))

	// ErrDeckHasRecords は紐づく Record があり Deck を削除できない場合(409)。
	ErrDeckHasRecords = New(http.StatusConflict, errors.New("cannot delete deck with records"))

	// ErrDeckCodeHasRecords は紐づく Record があり DeckCode を削除できない場合(409)。
	ErrDeckCodeHasRecords = New(http.StatusConflict, errors.New("cannot delete deckcode with records"))

	// ErrUserPlayerLocked は紐付けから1ヶ月経過しておらず変更できない場合(409)。
	ErrUserPlayerLocked = New(http.StatusConflict, errors.New("cannot change player_id within 1 month of linking"))

	// ErrTooManyPushSubscriptions は1ユーザーの push 購読(端末)数が上限に達している場合(409)。
	ErrTooManyPushSubscriptions = New(http.StatusConflict, errors.New("too many push subscriptions"))

	// ErrTooManyUserGyms は1ユーザーのMyジム登録数が上限に達している場合(409)。
	ErrTooManyUserGyms = New(http.StatusConflict, errors.New("too many user gyms"))

	// ErrTooManyRequests は短時間に試行が集中し、レート制限に達した場合(429)。
	ErrTooManyRequests = New(http.StatusTooManyRequests, errors.New("too many requests"))

	// ErrUserPlayerLinkingDisabled はプレイヤーID連携機能が運用者によって
	// 一時的に無効化されている場合(503)。
	ErrUserPlayerLinkingDisabled = New(http.StatusServiceUnavailable, errors.New("player id linking is currently disabled"))

	// ErrInternalServerError はサーバ内部エラー(500)。
	ErrInternalServerError = New(http.StatusInternalServerError, errors.New("internal server error"))

	// ErrBadGateway は上流サーバから不正な応答を受けた場合(502)。
	ErrBadGateway = New(http.StatusBadGateway, errors.New("bad gateway"))

	// ErrServiceUnavailable は依存サービスが利用できない場合(503)。
	ErrServiceUnavailable = New(http.StatusServiceUnavailable, errors.New("service unavailable"))

	// ErrGatewayTimeout は上流サーバが時間内に応答しない場合(504)。
	ErrGatewayTimeout = New(http.StatusGatewayTimeout, errors.New("gateway timeout"))
)
