package authorization

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// パスの :id と認証済み uid が一致する本人だけを通す。
//
// 戦績・連続記録・バッジ・称号・環境バッジは本人の活動記録そのもので、他人向けの画面は無い。
// uid はみんなの公開デッキの投稿者ページの URL で公開されているため、ここで絞らないと
// 誰でも他人の勝率や活動日数を uid 指定で読めてしまう(2026-09 の監査で指摘)。
// nginx はこのサーバを /api/v1beta で直接公開しているので、webapp(BFF)側の判定だけでは足りない。
func selfOnly() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" || uid != id {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func UserStatAuthorizationMiddleware() gin.HandlerFunc {
	return selfOnly()
}

func StreakAuthorizationMiddleware() gin.HandlerFunc {
	return selfOnly()
}

func BadgeAuthorizationMiddleware() gin.HandlerFunc {
	return selfOnly()
}

func DesignationAuthorizationMiddleware() gin.HandlerFunc {
	return selfOnly()
}

func EnvironmentBadgeAuthorizationMiddleware() gin.HandlerFunc {
	return selfOnly()
}
