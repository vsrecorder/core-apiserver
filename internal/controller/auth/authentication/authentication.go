package authentication

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

const (
	TokenLifetimeSecond = time.Duration(15) * time.Second
	ExpectedIssuer      = "vsrecorder-webapp"
)

type VSRClaims struct {
	jwt.RegisteredClaims
	UID string `json:"uid"`
}

/*
 * UserVerifier は、トークンの uid が登録済みで退会していないユーザーかを答える。
 *
 * トークンの署名と有効期限だけでは「そのユーザーが今も存在する」ことは保証されない。
 * 退会(論理削除)後も Firebase 側のアカウントが残ることがあり、そのトークンで書き込めると
 * 退会後にデータが作られてしまう。そこで認証ミドルウェアは署名の検証に加えて、
 * 実装(usecase.ActiveUserVerifier)へ uid の状態を問い合わせる。
 *
 * 実装は main が SetUserVerifier で注入する。未設定のまま認証が必要なルートへ来た場合は
 * 500 で止める(検査を黙って省略する fail open にしない)。
 */
type UserVerifier interface {
	IsActiveUser(ctx context.Context, uid string) (bool, error)
}

// UserVerifierFunc は関数を UserVerifier として使うための型(主にテスト用)。
type UserVerifierFunc func(ctx context.Context, uid string) (bool, error)

func (f UserVerifierFunc) IsActiveUser(ctx context.Context, uid string) (bool, error) {
	return f(ctx, uid)
}

var userVerifier UserVerifier

// SetUserVerifier は uid の状態を問い合わせる先を設定する。main が起動時に呼ぶ。
func SetUserVerifier(v UserVerifier) {
	userVerifier = v
}

var errUserVerifierNotConfigured = errors.New("user verifier is not configured")

func parseToken(tokenString string, secretKey string) (*jwt.Token, error) {
	// 鍵が空のまま検証すると、空鍵([]byte(""))で署名された偽造トークンを
	// 正当なものとして受け入れてしまう。起動時にも設定を検証しているが
	// (cmd/core-apiserver)、認証が丸ごと破れる影響の大きさを踏まえ、
	// 検証を行うこの場所でも必ず失敗させる。
	if secretKey == "" {
		return nil, errors.New("jwt secret is not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &VSRClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secretKey), nil
	},
		jwt.WithIssuer(ExpectedIssuer),
		// expを持たないトークンは有効期限が無いものとして通ってしまうため、
		// 発行側(webapp)がexpを付け忘れた場合や、漏洩したトークンが失効しない
		// 事態を防ぐために必須とする。
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// authenticate は Authorization ヘッダのトークンを検証して uid を返す。
// 失敗時は 401 を書き込んで ok=false を返す(呼び出し側はそのまま return する)。
func authenticate(ctx *gin.Context) (uid string, ok bool) {
	secretKey := os.Getenv("VSRECORDER_JWT_SECRET")

	header := http.Header{}
	header.Add("Authorization", ctx.GetHeader("Authorization"))

	tokenString := strings.TrimPrefix(header.Get("Authorization"), "Bearer ")

	token, err := parseToken(tokenString, secretKey)
	if err != nil {
		apierror.ErrUnauthorized.JSON(ctx, err)
		return "", false
	}

	claims := token.Claims.(*VSRClaims)

	if claims.UID == "" {
		apierror.ErrUnauthorized.JSON(ctx)
		return "", false
	}

	return claims.UID, true
}

// verifyActiveUser は uid が登録済みで退会していないことを確認する。
// 未登録・退会済みは 401、確認できない(未設定・DBエラー)場合は 500 を書き込んで false を返す。
func verifyActiveUser(ctx *gin.Context, uid string) bool {
	if userVerifier == nil {
		apierror.ErrInternalServerError.JSON(ctx, errUserVerifierNotConfigured)
		return false
	}

	active, err := userVerifier.IsActiveUser(ctx.Request.Context(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return false
	}

	if !active {
		apierror.ErrUnregisteredUser.JSON(ctx)
		return false
	}

	return true
}

// RequiredAuthenticationMiddleware は有効なトークンと、登録済みで退会していない uid を要求する。
func RequiredAuthenticationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uid, ok := authenticate(ctx)
		if !ok {
			return
		}

		// 先に uid を context へ載せ、拒否した場合のログにも uid が付くようにする
		// (中断するので後続のハンドラには進まない)。
		helper.SetUID(ctx, uid)

		if !verifyActiveUser(ctx, uid) {
			return
		}
	}
}

// RegistrationAuthenticationMiddleware はユーザー登録(POST /users)専用。
// 登録前の uid はまだ users に無いため、トークンの検証だけを行い、登録済みかは確認しない。
// 登録の可否(登録済み・退会済み)は usecase.User.Create が判定する。他のルートでは使わない。
func RegistrationAuthenticationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uid, ok := authenticate(ctx)
		if !ok {
			return
		}

		helper.SetUID(ctx, uid)
	}
}

// OptionalAuthenticationMiddleware はトークン無しを許容し(uid は空文字)、
// トークンがあれば RequiredAuthenticationMiddleware と同じ検証を行う。
// 退会済み・未登録の uid のトークンは、公開の一覧を返すためであっても 401 にする
// (「退会したのに使えている」状態を作らない。webapp は 401 でサインアウトへ誘導する)。
func OptionalAuthenticationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.GetHeader("Authorization") == "" {
			helper.SetUID(ctx, "")
			return
		}

		uid, ok := authenticate(ctx)
		if !ok {
			return
		}

		helper.SetUID(ctx, uid)

		if !verifyActiveUser(ctx, uid) {
			return
		}
	}
}
