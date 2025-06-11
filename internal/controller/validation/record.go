package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func RecordGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := helper.ParseQueryLimit(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		offset, err := helper.ParseQueryOffset(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		cursor, err := helper.ParseQueryCursor(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetLimit(ctx, limit)
		helper.SetOffset(ctx, offset)
		helper.SetCursor(ctx, cursor)
	}
}

func RecordCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.RecordCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		/*
			以下のデータで2つ以上値がある場合は bad request
			データが一つもない場合も bad request
			req.OfficialEventId
			req.TonamelEventId
			req.FriendId
		*/
		if req.OfficialEventId != 0 {
			if !(req.TonamelEventId == "" && req.FriendId == "") {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		} else if req.TonamelEventId != "" {
			if !(req.OfficialEventId == 0 && req.FriendId == "") {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		} else if req.FriendId != "" {
			if !(req.OfficialEventId == 0 && req.TonamelEventId == "") {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetRecordCreateRequest(ctx, req)
	}
}

func RecordUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.RecordUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		/*
			以下のデータで2つ以上値がある場合は bad request
			データが一つもない場合も bad request
			req.OfficialEventId
			req.TonamelEventId
			req.FriendId
		*/
		if req.OfficialEventId != 0 {
			if !(req.TonamelEventId == "" && req.FriendId == "") {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		} else if req.TonamelEventId != "" {
			if !(req.OfficialEventId == 0 && req.FriendId == "") {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		} else if req.FriendId != "" {
			if !(req.OfficialEventId == 0 && req.TonamelEventId == "") {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetRecordUpdateRequest(ctx, req)
	}
}
