package validation

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// isValidOpponentDeckSpec は表記の長さとスプライトの指定を、対戦結果の保存と同じ基準で検証する。
func isValidOpponentDeckSpec(spec *dto.OpponentDeckSpecRequest) bool {
	if spec == nil {
		return false
	}

	if exceedsLength(spec.OpponentsDeckInfo, MaxOpponentsDeckInfoLength) {
		return false
	}

	return validatePokemonSprites(spec.PokemonSprites)
}

/*
 * OpponentDeckReplaceMiddleware は PUT /matches/opponent_decks のボディを検証する。
 *
 * 置き換え元は表記かスプライトのどちらかを持つこと。どちらも空の指定を通すと、不戦勝・不戦敗など
 * 相手デッキの無い対戦がまとめて置き換わってしまう(一覧にも出していない組み合わせ)。
 * 置き換え先は表記を必須にする。表記を空にする一括操作は「まとめて消す」に近く、取り返しが
 * 付かないうえ、一括編集の用途(表記ゆれを揃える)に要らないため。
 */
func OpponentDeckReplaceMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.OpponentDeckReplaceRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if !isValidOpponentDeckSpec(req.From) || !isValidOpponentDeckSpec(req.To) {
			apierror.ErrBadRequest.JSON(ctx, errors.New("invalid opponent deck spec"))
			return
		}

		if req.From.OpponentsDeckInfo == "" && len(req.From.PokemonSprites) == 0 {
			apierror.ErrBadRequest.JSON(ctx, errors.New("from must have opponents_deck_info or pokemon_sprites"))
			return
		}

		if strings.TrimSpace(req.To.OpponentsDeckInfo) == "" {
			apierror.ErrBadRequest.JSON(ctx, errors.New("to.opponents_deck_info is required"))
			return
		}

		helper.SetOpponentDeckReplaceRequest(ctx, req)
	}
}
