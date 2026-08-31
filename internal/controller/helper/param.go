package helper

import (
	"github.com/gin-gonic/gin"
)

func GetId(ctx *gin.Context) (id string) {
	return ctx.Param("id")
}

// GetParamShopId はパスの :shop_id を返す。Myジムの解除
// (DELETE /users/my_gyms/:shop_id)で使う。
func GetParamShopId(ctx *gin.Context) string {
	return ctx.Param("shop_id")
}
