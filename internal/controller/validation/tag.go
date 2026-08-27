package validation

import (
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// tagColorPattern は '#RRGGBB' 形式(16進6桁)を表す。色は任意項目のため空文字も許容する。
var tagColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// isValidTagColor はタグの色として受け入れられる値かを確認する。
// 空文字(未設定)か '#RRGGBB' のみを許可する。webapp が chip の背景色にそのまま
// 使うため、想定外の値(javascript: など)を保存させない。
func isValidTagColor(s string) bool {
	if s == "" {
		return true
	}
	return tagColorPattern.MatchString(s)
}

// TagGetPresetsMiddleware は GET /tags/presets の category クエリ(プリセットの群)を
// 解釈して context へ入れる。未定義の値は空文字(全プリセット)へ丸めるため 400 は返さない。
func TagGetPresetsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		helper.SetTagPresetCategory(ctx, helper.ParseQueryTagPresetCategory(ctx))
	}
}

func TagCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.TagCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if req.Name == "" || exceedsLength(req.Name, MaxTagNameLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidTagColor(req.Color) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetTagCreateRequest(ctx, req)
	}
}

func TagUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.TagUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if req.Name == "" || exceedsLength(req.Name, MaxTagNameLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidTagColor(req.Color) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetTagUpdateRequest(ctx, req)
	}
}

// validateTagIds は付与先(デッキ/デッキコード/記録/対戦結果)のリクエストに
// 埋め込まれた tag_ids を検証する。
// 個々のIDの実在・所有権チェックは usecase 層(所有者で絞り込み)に任せ、ここでは
// 件数と各IDの長さ(ULID=26文字)だけを確認して極端な値を弾く。
func validateTagIds(tagIds []string) bool {
	if len(tagIds) > MaxTagsPerEntity {
		return false
	}
	for _, id := range tagIds {
		if id == "" || exceedsLength(id, 26) {
			return false
		}
	}
	return true
}
