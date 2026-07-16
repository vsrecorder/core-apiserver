package authorization

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// CalendarAuthorizationMiddleware はカレンダーを本人しか取得できないようにする。
// カレンダーは記録・デッキ・対戦結果を丸ごと含むため、他人のIDを指定された場合は必ず弾く。
func CalendarAuthorizationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		if uid != id {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}
