// Package apierror はコントローラ層で扱う、HTTPステータスコードを内包した
// 独自エラーを定義する。
//
// ドメイン層のエラー(apperror)が「何が起きたか」を表すのに対し、apierror は
// 「クライアントへどのステータス・メッセージで応答するか」という HTTP 表現を
// 担う。各ハンドラは応答内容を文字列リテラルで直接書く代わりに、ここで定義した
// 値を JSON(ctx) で返すことで、ステータスとメッセージの対応を一元管理できる。
package apierror

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
// 後続ハンドラの実行を中断する。
func (e *Error) JSON(ctx *gin.Context) {
	ctx.JSON(e.status, gin.H{"message": e.err.Error()})
	ctx.Abort()
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

	// ErrDeckHasRecords は紐づく Record があり Deck を削除できない場合(409)。
	ErrDeckHasRecords = New(http.StatusConflict, errors.New("cannot delete deck with records"))

	// ErrDeckCodeHasRecords は紐づく Record があり DeckCode を削除できない場合(409)。
	ErrDeckCodeHasRecords = New(http.StatusConflict, errors.New("cannot delete deckcode with records"))

	// ErrUserPlayerLocked は紐付けから1ヶ月経過しておらず変更できない場合(409)。
	ErrUserPlayerLocked = New(http.StatusConflict, errors.New("cannot change player_id within 1 month of linking"))

	// ErrPlayerIdAlreadyLinked は指定された player_id が既に別のユーザーに紐付けられている場合(409)。
	ErrPlayerIdAlreadyLinked = New(http.StatusConflict, errors.New("this player_id is already linked to another account"))

	// ErrUserPlayerInvalidChallenge は所有権確認チャレンジのトークンが不正・
	// 期限切れ、または発行時と異なるユーザー/player_idに対して使われた場合(400)。
	ErrUserPlayerInvalidChallenge = New(http.StatusBadRequest, errors.New("invalid or expired ownership challenge, please try again from the beginning"))

	// ErrUserPlayerOwnershipNotVerified はアバター画像がチャレンジで指定した
	// ものに変更されていることを確認できなかった場合(403)。
	ErrUserPlayerOwnershipNotVerified = New(http.StatusForbidden, errors.New("could not verify that the avatar image has been changed as requested"))

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
