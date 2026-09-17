package controller

import (
	"context"
	"os"
	"testing"

	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
)

func TestMain(m *testing.M) {
	// 認証ミドルウェアはトークンの uid が登録済みで退会していないかを検査する
	// (authentication.SetUserVerifier で注入)。コントローラのテストは任意の uid を
	// 使うため、常に有効なユーザーとして通す。検査そのものは authentication のテストが受け持つ。
	authentication.SetUserVerifier(authentication.UserVerifierFunc(func(context.Context, string) (bool, error) {
		return true, nil
	}))

	os.Exit(m.Run())
}
