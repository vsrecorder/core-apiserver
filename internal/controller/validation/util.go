package validation

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/httpclient"
)

const (
	DateLayout = time.DateOnly
)

// DeckIDCheckURL はデッキコードの実在確認APIのURL。外部サイトへ実通信せずに
// テストできるよう、httptestサーバへ差し替え可能な変数にしている。
var DeckIDCheckURL = "https://www.pokemon-card.com/deck/deckIDCheck.php"

// 文字列長の上限。VARCHARのカラムは db/schema.sql の定義と対応させており、
// 超過した値がPostgres側のエラー(500)になる前に400として弾く。
// memo等のTEXTカラムはDB側に上限が無いため、実用上十分な値をここで定める。
const (
	MaxUserNameLength = 63  // users.name VARCHAR(63)
	MaxImageURLLength = 255 // users.image_url VARCHAR(255)

	MaxDeckNameLength = 32 // decks.name VARCHAR(32)
	MaxDeckCodeLength = 21 // deck_codes.code VARCHAR(21)

	MaxEventTitleLength = 255 // unofficial_events.title VARCHAR(255)

	MaxOpponentsDeckInfoLength = 63 // matches.opponents_deck_info VARCHAR(63)

	MaxMemoLength = 10000 // deck_codes.memo / records.memo / matches.memo / games.memo (TEXT)
	MaxURLLength  = 2048  // records.tcg_meister_url (TEXT)
)

// isValidImageURL は画像URLとして受け入れられる値かを確認する。
//
// スキームを検証しない場合 javascript: や data: をそのまま保存でき、
// GET /users/:id は認証不要で誰でも取得できるため、描画側の実装次第では
// XSSに繋がる。またhttp:を許すと閲覧者の通信内容が経路上に漏れる。
// 正規の値(デフォルトアイコン/CDNへのアップロード結果)はいずれもhttpsのため、
// httpsのみに限定しておけば描画側の実装によらず安全側に倒せる。
func isValidImageURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// url.Parseはスキームを小文字に正規化するため、大文字混じりの
	// "JavaScript:" のような値もここで弾ける。
	if u.Scheme != "https" {
		return false
	}

	// "https:/path" のようにホストを持たない値を除く。
	if u.Host == "" {
		return false
	}

	return true
}

// isValidTCGMeisterURL はrecords.tcg_meister_urlとして受け入れられる値かを確認する。
//
// この項目は任意入力のため、未設定(空文字)は許容する。
// 値がある場合にスキームを検証しないと javascript: や data: をそのまま保存でき、
// webappはこの値を <a href> にそのまま入れて描画するためXSSに繋がる。
// 記録は本人にしか表示されず影響は自己XSSに留まるが、描画側の実装次第で安全性が
// 決まる状態にはしない。
//
// webapp側にも入力チェックはあるが、あれは送信ボタンの活性を切り替えるだけで、
// APIを直接叩けば素通りする。保存を受け付けるここが実際の防御線になる。
func isValidTCGMeisterURL(s string) bool {
	if s == "" {
		return true
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// url.Parseはスキームを小文字に正規化するため、大文字混じりの
	// "JavaScript:" のような値もここで弾ける。
	// 外部サイトへのリンクであり、webappの入力欄がhttp/httpsの両方を
	// 受け付けているため、ここでも両方を許容する。
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// "https:/path" のようにホストを持たない値を除く。
	if u.Host == "" {
		return false
	}

	return true
}

// exceedsLength は文字列がmax文字を超えているかを返す。
//
// PostgresのVARCHAR(n)はバイト数ではなく文字数で制限するため、len()ではなく
// ルーン数で数える。デッキ名やメモは日本語が主であり、バイト数で判定すると
// スキーマ上は収まる文字列を誤って拒否してしまう。
func exceedsLength(s string, max int) bool {
	return utf8.RuneCountInString(s) > max
}

type DeckIDCheckResponse struct {
	Result    int    `json:"result"`
	ErrMsg    string `json:"errMsg"`
	DeckID    string `json:"deckID"`
	Existence int    `json:"existence"`
}

// deckCodeCheckLogAttrs はデッキコード確認APIのログに共通で付与する属性を返す。
// request_idはアクセスログと突き合わせるためにRequestIDMiddlewareが設定した値を利用する。
func deckCodeCheckLogAttrs(ctx *gin.Context, deckCode string) []any {
	requestID, _ := ctx.Get("request_id")
	requestIDStr, _ := requestID.(string)

	return []any{
		slog.String("request_id", requestIDStr),
		slog.String("deck_code", deckCode),
		slog.String("request_url", DeckIDCheckURL),
	}
}

func checkDeckCode(ctx *gin.Context, logger *slog.Logger, deckCode string) {
	data := url.Values{}
	data.Add("deckID", deckCode)

	resp, err := httpclient.PostForm(DeckIDCheckURL, data)

	if err != nil {
		logger.ErrorContext(
			ctx.Request.Context(),
			"failed to request deck code check API",
			append(
				deckCodeCheckLogAttrs(ctx, deckCode),
				slog.String("error_message", err.Error()),
			)...,
		)

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorContext(
			ctx.Request.Context(),
			"deck code check API returned non-200 status",
			append(
				deckCodeCheckLogAttrs(ctx, deckCode),
				slog.Int("status_code", resp.StatusCode),
			)...,
		)

		switch resp.StatusCode {
		case http.StatusServiceUnavailable:
			apierror.ErrServiceUnavailable.JSON(ctx)
			return
		case http.StatusGatewayTimeout:
			apierror.ErrGatewayTimeout.JSON(ctx)
			return
		default:
			apierror.ErrBadGateway.JSON(ctx)
			return
		}
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	var res DeckIDCheckResponse
	if err := json.Unmarshal(body, &res); err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	if res.Existence == 0 {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}
}
