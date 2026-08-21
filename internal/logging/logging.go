// Package logging は全レイヤーで共通して使うログ出力の土台を提供する。
//
// ログを1リクエスト単位で追えるようにするには request_id が全レイヤーのログへ
// 付いている必要があるが、各レイヤーが *slog.Logger や request_id を引数で
// 引き回すのはクリーンアーキテクチャの依存方向を汚す上に、全コンストラクタの
// シグネチャ変更を強いる。そこで request_id / uid は context に載せ、
// ContextHandler が全てのログレコードへ自動で付与する方式を採っている。
// 各レイヤーは ctx を渡して slog の *Context 系メソッドを呼ぶだけでよい。
package logging

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// レイヤー名。ログの layer 属性に入れ、どの層で発生したログかを絞り込めるようにする。
const (
	LayerController     = "controller"
	LayerUsecase        = "usecase"
	LayerInfrastructure = "infrastructure"
)

// contextKey は context へ値を出し入れするための非公開キー型。
// 文字列キーだと他パッケージの値と衝突しうるため独自型にしている。
type contextKey int

const (
	requestIDKey contextKey = iota
	uidKey
)

// ContextWithRequestID は request_id を載せた context を返す。
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext は context から request_id を取り出す。無い場合は空文字を返す。
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	requestID, _ := ctx.Value(requestIDKey).(string)

	return requestID
}

// ContextWithUID は uid を載せた context を返す。
func ContextWithUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, uidKey, uid)
}

// UIDFromContext は context から uid を取り出す。無い場合は空文字を返す。
func UIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	uid, _ := ctx.Value(uidKey).(string)

	return uid
}

// ContextHandler は slog.Handler をラップし、context が持つ request_id / uid を
// 全てのログレコードへ自動付与する。
//
// これを挟むことで、usecase や infrastructure は request_id の存在を知らないまま
// アクセスログと同じ request_id が付いたログを出力できる。
type ContextHandler struct {
	slog.Handler
}

// NewContextHandler は handler をラップした ContextHandler を返す。
func NewContextHandler(handler slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: handler}
}

// Handle は context の request_id / uid を属性へ加えてから元のハンドラへ委譲する。
func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		record.AddAttrs(slog.String("request_id", requestID))
	}

	if uid := UIDFromContext(ctx); uid != "" {
		record.AddAttrs(slog.String("uid", uid))
	}

	return h.Handler.Handle(ctx, record)
}

// WithAttrs はラップを維持したまま属性を追加したハンドラを返す。
// ラップを外すと With() 以降のログから request_id が消えるため、必ず包み直す。
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup はラップを維持したままグループを開いたハンドラを返す。
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// Err はエラーを全レイヤー共通のキーでログへ載せる。
// nil を渡しうる箇所でも安全に使えるようにしている。
func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error_message", "")
	}

	return slog.String("error_message", err.Error())
}

// Operation はログの発生箇所(ハンドラ名・ユースケース名・リポジトリメソッド名)を示す。
// "Record.Create" のように リソース名.メソッド名 の形式で渡す。
func Operation(name string) slog.Attr {
	return slog.String("operation", name)
}

// Layered は layer 属性を固定したロガーを返す。
//
// slog.Default() を呼び出しの都度参照するため、main が slog.SetDefault() を
// 呼ぶ前にこの関数の戻り値をパッケージ変数へ束縛しない限り、常に設定済みの
// ハンドラが使われる。
func Layered(layer string) *slog.Logger {
	return slog.Default().With(slog.String("layer", layer))
}

// LogAt は「実際の呼び出し元」の位置をソース情報として記録しつつログを出力する。
//
// 共通ヘルパー(apierror.JSON や infrastructure.wrapError)の中から slog の通常の
// メソッドを呼ぶと、ソース位置がヘルパー自身になってしまい、どのハンドラ・どの
// クエリで起きたのかが追えなくなる。skip 段上のフレームの PC を使うことでこれを
// 避けている。skip=0 は LogAt の呼び出し元を指す。
func LogAt(
	ctx context.Context,
	logger *slog.Logger,
	level slog.Level,
	skip int,
	msg string,
	attrs ...slog.Attr,
) {
	if !logger.Enabled(ctx, level) {
		return
	}

	// runtime.Callers の skip は 0 が runtime.Callers 自身、1 が LogAt 自身。
	var pcs [1]uintptr
	runtime.Callers(skip+2, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	record.AddAttrs(attrs...)

	// logger.Handler() には logger.With() で付与済みの属性が含まれるため、
	// ここで組み立てたレコードでも layer などの属性は失われない。
	_ = logger.Handler().Handle(ctx, record)
}

// CallerOperation は skip 段上の呼び出し元の関数名を operation 用の文字列で返す。
// skip=0 は CallerOperation の呼び出し元を指す。
// パッケージパスは冗長なため、最後の要素("controller.(*Record).GetById")だけを残す。
func CallerOperation(skip int) string {
	var pcs [1]uintptr
	if runtime.Callers(skip+2, pcs[:]) == 0 {
		return ""
	}

	frame, _ := runtime.CallersFrames(pcs[:]).Next()

	name := frame.Function
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	return name
}
