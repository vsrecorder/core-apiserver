package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	DecksPath = "/decks"
)

type Deck struct {
	router           *gin.Engine
	deckRepository   repository.DeckInterface
	recordRepository repository.RecordInterface
	usecase          usecase.DeckInterface
}

func NewDeck(
	router *gin.Engine,
	deckRepository repository.DeckInterface,
	recordRepository repository.RecordInterface,
	usecase usecase.DeckInterface,
) *Deck {
	return &Deck{router, deckRepository, recordRepository, usecase}
}

func (c *Deck) RegisterRoute(relativePath string, authDisable bool) {
	if authDisable {
		r := c.router.Group(relativePath + DecksPath)
		r.GET(
			"",
			validation.DeckGetMiddleware(),
			c.Get,
			c.GetByUserId,
		)
		r.GET(
			"/all",
			c.GetAll,
		)
		r.GET(
			"/:id",
			c.GetById,
		)
		r.POST(
			"",
			validation.DeckCreateMiddleware(),
			c.Create,
		)
		r.PUT(
			"/:id",
			validation.DeckUpdateMiddleware(),
			c.Update,
		)
		r.PATCH(
			"/:id/archive",
			c.Archive,
		)
		r.PATCH(
			"/:id/unarchive",
			c.Unarchive,
		)
		r.DELETE(
			"/:id",
			c.Delete,
		)
	} else {
		r := c.router.Group(relativePath + DecksPath)
		r.GET(
			"",
			authentication.OptionalAuthenticationMiddleware(),
			validation.DeckGetMiddleware(),
			c.Get,
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
			validation.DeckCreateMiddleware(),
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
		r.DELETE(
			"/:id",
			authentication.RequiredAuthenticationMiddleware(),
			authorization.DeckDeleteAuthorizationMiddleware(c.deckRepository, c.recordRepository),
			c.Delete,
		)
	}
}

func (c *Deck) Get(ctx *gin.Context) {
	if uid := helper.GetUID(ctx); uid == "" {
		limit := helper.GetLimit(ctx)
		offset := helper.GetOffset(ctx)
		cursor := helper.GetCursor(ctx)

		if !cursor.IsZero() {
			decks, err := c.usecase.FindOnCursor(context.Background(), limit, cursor)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
				return
			}

			for _, deck := range decks {
				if deck.LatestDeckCode.PrivateCodeFlg {
					deck.LatestDeckCode.Code = ""
				}
			}

			res := presenter.NewDeckGetResponse(limit, offset, cursor, decks)

			ctx.JSON(http.StatusOK, res)
		} else {
			decks, err := c.usecase.Find(context.Background(), limit, offset)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
				return
			}

			for _, deck := range decks {
				if deck.LatestDeckCode.PrivateCodeFlg {
					deck.LatestDeckCode.Code = ""
				}
			}

			res := presenter.NewDeckGetResponse(limit, offset, cursor, decks)

			ctx.JSON(http.StatusOK, res)
		}
	}
}

func (c *Deck) GetAll(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	decks, err := c.usecase.FindAll(context.Background(), uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
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
			decks, err := c.usecase.FindByUserIdOnCursor(context.Background(), uid, archived, limit, cursor)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
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
			decks, err := c.usecase.FindByUserId(context.Background(), uid, archived, limit, offset)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
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

	deck, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
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

	param := usecase.NewDeckCreateParam(
		uid,
		req.Name,
		req.PrivateFlg,
		req.DeckCode,
		req.PrivateDeckCodeFlg,
	)

	deck, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
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
		pokemonSprites = append(pokemonSprites, usecase.NewPokemonSpriteParam(pokemonSprite.ID))
	}

	param := usecase.NewDeckUpdateParam(
		req.Name,
		req.PrivateFlg,
		pokemonSprites,
	)

	deck, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewDeckUpdateResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Archive(ctx *gin.Context) {
	id := helper.GetId(ctx)

	deck, err := c.usecase.Archive(context.Background(), id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewDeckArchiveResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Unarchive(ctx *gin.Context) {
	id := helper.GetId(ctx)

	deck, err := c.usecase.Unarchive(context.Background(), id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewDeckUnarchiveResponse(deck)

	ctx.JSON(http.StatusOK, res)
}

func (c *Deck) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(context.Background(), id); err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
