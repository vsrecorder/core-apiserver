package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func RecordGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := helper.ParseQueryLimit(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		offset, err := helper.ParseQueryOffset(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		cursorEventDate, cursorCreatedAt, err := helper.ParseQueryCursor(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		eventType, err := helper.ParseQueryEventType(ctx)

		helper.SetLimit(ctx, limit)
		helper.SetOffset(ctx, offset)
		helper.SetCursorEventDate(ctx, cursorEventDate)
		helper.SetCursorCreatedAt(ctx, cursorCreatedAt)
		helper.SetEventType(ctx, eventType)

		helper.SetDeckId(ctx, helper.GetQueryDeckId(ctx))
	}
}

func RecordCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.RecordCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidRecordEventSource(req.RecordRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidRecordLength(req.RecordRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidTCGMeisterURL(req.RecordRequest.TCGMeisterURL) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetRecordCreateRequest(ctx, req)
	}
}

func RecordUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.RecordUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidRecordEventSource(req.RecordRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidRecordLength(req.RecordRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidTCGMeisterURL(req.RecordRequest.TCGMeisterURL) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetRecordUpdateRequest(ctx, req)
	}
}

/*
記録の紐づくイベントは以下の4種類のうち、ちょうど1つだけ指定されている必要がある。
2つ以上指定されている場合も、1つも指定されていない場合も bad request とする。
  - 公式イベント   : OfficialEventId
  - Tonamel       : TonamelEventId
  - フレンド対戦   : FriendId
  - 自由形式       : UnofficialEventId
*/
func isValidRecordEventSource(req dto.RecordRequest) bool {
	count := 0
	if req.OfficialEventId != 0 {
		count++
	}
	if req.TonamelEventId != "" {
		count++
	}
	if req.FriendId != "" {
		count++
	}
	if req.UnofficialEventId != "" {
		count++
	}

	if count != 1 {
		return false
	}

	return true
}

// isValidRecordLength は自由入力欄が上限内に収まっているかを確認する。
// memo と tcg_meister_url はDB上TEXTで上限が無いため、ここで歯止めをかける。
func isValidRecordLength(req dto.RecordRequest) bool {
	if exceedsLength(req.Memo, MaxMemoLength) {
		return false
	}

	if exceedsLength(req.TCGMeisterURL, MaxURLLength) {
		return false
	}

	return true
}
