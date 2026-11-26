package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	DecksPath = "/decks"
)

type Deck struct {
	logger           *slog.Logger
	router           *gin.Engine
	deckRepository   repository.DeckInterface
	recordRepository repository.RecordInterface
	usecase          usecase.DeckInterface
}

func NewDeck(
	logger *slog.Logger,
	router *gin.Engine,
	deckRepository repository.DeckInterface,
	recordRepository repository.RecordInterface,
	usecase usecase.DeckInterface,
) *Deck {
	return &Deck{logger, router, deckRepository, recordRepository, usecase}
}

func (c *Deck) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + DecksPath)
	//
	// 一覧は本人のデッキだけを返す。ログインしていなければ 401。
	//
	// 以前は未ログインの要求に、全ユーザーの公開設定(private_flg = false)のデッキを、
	// ユーザー ID・最新デッキコードのメモごと返していた。webapp にはこの一覧を使う画面が無く
	// (BFF の /api/decks もデッキ一覧ページのサーバ側の取得もトークン付き)、nginx が /api/v1beta を
	// そのまま公開しているため、製品のどこからも使われない口から誰でも一覧できた
	// (2026-10-31 のセキュリティ精査。記録の一覧(M-4)と同じ作り)。
	// 他人のデッキを見せる機能を作るときは、ここを開け直すのではなく、返す項目を明示的に絞った
	// 専用の API を別に用意すること(みんなの公開デッキは deck_code_posts がそれにあたる)。
	//
	r.GET(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.DeckGetMiddleware(),
		c.GetByUserId,
	)
	r.GET(
		"/all",
		authentication.RequiredAuthenticationMiddleware(),
		c.GetAll,
	)
	r.GET(
		"/:id",
		authentication.OptionalAuthenticationMiddleware(),
		authorization.DeckGetByIdAuthorizationMiddleware(c.deckRepository),
		c.GetById,
	)
	r.POST(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.DeckCreateMiddleware(c.logger),
		c.Create,
	)
	r.PUT(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckUpdateAuthorizationMiddleware(c.deckRepository),
		validation.DeckUpdateMiddleware(),
		c.Update,
	)
	r.PATCH(
		"/:id/archive",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckArchiveAuthorizationMiddleware(c.deckRepository),
		c.Archive,
	)
	r.PATCH(
		"/:id/unarchive",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckUnarchiveAuthorizationMiddleware(c.deckRepository),
		c.Unarchive,
	)
	r.PATCH(
		"/:id/favorite",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckFavoriteAuthorizationMiddleware(c.deckRepository),
		c.Favorite,
	)
	r.PATCH(
		"/:id/unfavorite",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckUnfavoriteAuthorizationMiddleware(c.deckRepository),
		c.Unfavorite,
	)
	r.DELETE(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckDeleteAuthorizationMiddleware(c.deckRepository, c.recordRepository),
		c.Delete,
	)
}

func (c *Deck) GetAll(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	decks, err := c.usecase.FindAll(ctx.Request.Context(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckGetAllResponse(decks)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) GetByUserId(ctx *gin.Context) {
	if uid := helper.GetUID(ctx); uid != "" {
		archived := helper.GetArchived(ctx)
		limit := helper.GetLimit(ctx)
		offset := helper.GetOffset(ctx)
		cursor := helper.GetCursor(ctx)

		if !cursor.IsZero() {
			decks, err := c.usecase.FindByUserIdOnCursor(ctx.Request.Context(), uid, archived, limit, cursor)
			if err != nil {
				apierror.ErrInternalServerError.JSON(ctx, err)
				return
			}

			for _, deck := range decks {
				if deck.LatestDeckCode.PrivateCodeFlg && uid != deck.LatestDeckCode.UserId {
					deck.LatestDeckCode.Code = ""
				}
			}

			res := presenter.NewDeckGetByUserIdResponse(archived, limit, offset, cursor, decks)

			ctx.JSON(http.StatusOK, res)
		} else {
			decks, err := c.usecase.FindByUserId(ctx.Request.Context(), uid, archived, limit, offset)
			if err != nil {
				apierror.ErrInternalServerError.JSON(ctx, err)
				return
			}

			for _, deck := range decks {
				if deck.LatestDeckCode.PrivateCodeFlg && uid != deck.LatestDeckCode.UserId {
					deck.LatestDeckCode.Code = ""
				}
			}
			res := presenter.NewDeckGetByUserIdResponse(archived, limit, offset, cursor, decks)

			ctx.JSON(http.StatusOK, res)
		}
	}
}

func (c *Deck) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	deck, err := c.usecase.FindById(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	if deck.LatestDeckCode.PrivateCodeFlg && uid != deck.LatestDeckCode.UserId {
		deck.LatestDeckCode.Code = ""
	}

	res := presenter.NewDeckGetByIdResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Create(ctx *gin.Context) {
	req := helper.GetDeckCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	var pokemonSprites []*usecase.PokemonSpriteParam
	for _, pokemonSprite := range req.PokemonSprites {
		pokemonSprites = append(
			pokemonSprites,
			usecase.NewPokemonSpriteParamWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
	}

	param := usecase.NewDeckCreateParam(
		uid,
		req.Name,
		req.PrivateFlg,
		req.DeckCode,
		req.PrivateDeckCodeFlg,
		pokemonSprites,
		req.TagIds,
	)

	deck, err := c.usecase.Create(ctx.Request.Context(), param)
	if err != nil {
		// デッキコード取得元(ポケモンカード公式サイト)がメンテナンス中の場合は 503 を返す
		if errors.Is(err, apperror.ErrUnderMaintenance) {
			apierror.ErrServiceUnavailable.JSON(ctx, err)
			return
		}

		// 存在しないスプライトID等、参照先が無い入力(外部キー違反)は 400。
		if errors.Is(err, apperror.ErrInvalidReference) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckCreateResponse(deck)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Deck) Update(ctx *gin.Context) {
	req := helper.GetDeckUpdateRequest(ctx)
	id := helper.GetId(ctx)

	var pokemonSprites []*usecase.PokemonSpriteParam
	for _, pokemonSprite := range req.PokemonSprites {
		pokemonSprites = append(
			pokemonSprites,
			usecase.NewPokemonSpriteParamWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
	}

	param := usecase.NewDeckUpdateParam(
		req.Name,
		req.PrivateFlg,
		pokemonSprites,
		req.TagIds,
	)

	deck, err := c.usecase.Update(ctx.Request.Context(), id, param)
	if err != nil {
		// 存在しないスプライトID等、参照先が無い入力(外部キー違反)は 400。
		if errors.Is(err, apperror.ErrInvalidReference) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckUpdateResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Archive(ctx *gin.Context) {
	id := helper.GetId(ctx)

	deck, err := c.usecase.Archive(ctx.Request.Context(), id)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckArchiveResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Unarchive(ctx *gin.Context) {
	id := helper.GetId(ctx)

	deck, err := c.usecase.Unarchive(ctx.Request.Context(), id)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckUnarchiveResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Favorite(ctx *gin.Context) {
	id := helper.GetId(ctx)

	deck, err := c.usecase.Favorite(ctx.Request.Context(), id)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckFavoriteResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Unfavorite(ctx *gin.Context) {
	id := helper.GetId(ctx)

	deck, err := c.usecase.Unfavorite(ctx.Request.Context(), id)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckUnfavoriteResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(ctx.Request.Context(), id); err != nil {
		if err == apperror.ErrRecordNotFound {
			apierror.ErrBadRequestNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
