package helper

import (
	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

func SetLimit(ctx *gin.Context, value int) {
	ctx.Set("limit", value)
}

func GetLimit(ctx *gin.Context) int {
	value, _ := ctx.Get("limit")
	limit, _ := value.(int)

	return limit

}

func SetOffset(ctx *gin.Context, value int) {
	ctx.Set("offset", value)
}

func GetOffset(ctx *gin.Context) int {
	value, _ := ctx.Get("offset")
	offset, _ := value.(int)

	return offset
}

func SetUID(ctx *gin.Context, value string) {
	ctx.Set("uid", value)
}

func GetUID(ctx *gin.Context) string {
	value, _ := ctx.Get("uid")
	uid, _ := value.(string)

	return uid
}

func SetRecordCreateRequest(ctx *gin.Context, value dto.RecordCreateRequest) {
	ctx.Set("record_create_request", value)
}

func GetRecordCreateRequest(ctx *gin.Context) dto.RecordCreateRequest {
	value, _ := ctx.Get("record_create_request")
	recordRequest, _ := value.(dto.RecordCreateRequest)

	return recordRequest
}
