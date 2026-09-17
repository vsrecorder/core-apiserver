package authorization

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func DeckCodeAuthorizationMiddleware(repository repository.DeckCodeInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		deckcode, err := repository.FindById(ctx.Request.Context(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if uid != deckcode.UserId {
			apierror.ErrForbidden.JSON(ctx, err)
			return
		}
	}
}

func DeckCodeUpdateAuthorizationMiddleware(repository repository.DeckCodeInterface) gin.HandlerFunc {
	return DeckCodeAuthorizationMiddleware(repository)
}

// DeckCodeGetByIdAuthorizationMiddleware はデッキコードの参照を、親デッキの公開範囲に合わせる。
//
// デッキコードは単体では公開・非公開を持たず(private_code_flg はコード文字列を伏せるだけ)、
// 見せてよいかは親デッキの private_flg で決まる。デッキ本体(GET /decks/:id)は非公開なら
// 他人に 403 を返すのに、その派生であるデッキコード(メモ等)だけ読めてしまわないようにする。
// デッキIDは公開記録の deck_id から誰でも知り得るため、「IDを知らなければ辿れない」は
// 防御にならない。判定は MatchGetByIdAuthorizationMiddleware(対戦→記録)と同じ形。
func DeckCodeGetByIdAuthorizationMiddleware(deckcodeRepository repository.DeckCodeInterface, deckRepository repository.DeckInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		deckcode, err := deckcodeRepository.FindById(ctx.Request.Context(), id)
		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		deck, err := deckRepository.FindById(ctx.Request.Context(), deckcode.DeckId)
		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if deck.PrivateFlg && uid != deck.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func DeckCodeDeleteAuthorizationMiddleware(deckcodeRepository repository.DeckCodeInterface, recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		deckcode, err := deckcodeRepository.FindById(ctx.Request.Context(), id)
		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if uid != deckcode.UserId {
			apierror.ErrForbidden.JSON(ctx, err)
			return
		}

		// 「使用中」の判定は所有者自身の記録に限る(デッキの削除と同じ理由)。
		limit := 1
		offset := 0
		records, err := recordRepository.FindByDeckCodeId(ctx.Request.Context(), deckcode.UserId, id, limit, offset)
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if len(records) > 0 {
			apierror.ErrDeckCodeHasRecords.JSON(ctx, err)
			return
		}
	}
}
