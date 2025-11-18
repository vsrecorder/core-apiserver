package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	MatchesPath = "/matches"
)

type Match struct {
	router     *gin.Engine
	repository repository.MatchInterface
	usecase    usecase.MatchInterface
}

func NewMatch(
	router *gin.Engine,
	repository repository.MatchInterface,
	usecase usecase.MatchInterface,
) *Match {
	return &Match{router, repository, usecase}
}

func (c *Match) RegisterRoute(relativePath string, authDisable bool) {
	if authDisable {
		{
			r := c.router.Group(relativePath + MatchesPath)
			r.GET(
				"/:id",
				c.GetById,
			)
			r.POST(
				"",
				validation.MatchCreateMiddleware(),
				c.Create,
			)
			r.PUT(
				"/:id",
				validation.MatchUpdateMiddleware(),
				c.Update,
			)
			r.DELETE(
				"/:id",
				c.Delete,
			)
		}

		{
			r := c.router.Group(relativePath + RecordsPath)
			r.GET(
				"/:id"+MatchesPath,
				c.GetByRecordId,
			)
		}
	} else {
		{
			r := c.router.Group(relativePath + MatchesPath)
			r.GET(
				"/:id",
				c.GetById,
			)
			r.POST(
				"",
				auth.RequiredAuthenticationMiddleware(),
				validation.MatchCreateMiddleware(),
				c.Create,
			)
			r.PUT(
				"/:id",
				auth.RequiredAuthenticationMiddleware(),
				auth.MatchUpdateAuthorizationMiddleware(c.repository),
				validation.MatchUpdateMiddleware(),
				c.Update,
			)
			r.DELETE(
				"/:id",
				auth.RequiredAuthenticationMiddleware(),
				auth.MatchDeleteAuthorizationMiddleware(c.repository),
				c.Delete,
			)
		}

		{
			r := c.router.Group(relativePath + RecordsPath)
			r.GET(
				"/:id"+MatchesPath,
				c.GetByRecordId,
			)
		}
	}
}

func (c *Match) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	match, err := c.usecase.FindById(context.Background(), id)
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

	res := presenter.NewMatchGetByIdResponse(match)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) GetByRecordId(ctx *gin.Context) {
	recordId := helper.GetId(ctx)

	matches, err := c.usecase.FindByRecordId(context.Background(), recordId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchGetByRecordIdResponse{})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewMatchGetByRecordIdResponse(matches)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) Create(ctx *gin.Context) {
	req := helper.GetMatchCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	var games []*usecase.GameParam
	for _, gameReq := range req.Games {
		games = append(
			games,
			usecase.NewGameParam(
				gameReq.GoFirst,
				gameReq.WinningFlg,
				gameReq.YourPrizeCards,
				gameReq.OpponentsPrizeCards,
				gameReq.Memo,
			),
		)
	}

	param := usecase.NewMatchParam(
		req.RecordId,
		req.DeckId,
		uid,
		req.OpponentsUserId,
		req.BO3Flg,
		req.QualifyingRoundFlg,
		req.FinalTournamentFlg,
		req.DefaultVictoryFlg,
		req.DefaultDefeatFlg,
		req.VictoryFlg,
		req.OpponentsDeckInfo,
		req.Memo,
		games,
	)

	match, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewMatchCreateResponse(match)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Match) Update(ctx *gin.Context) {
	req := helper.GetMatchUpdateRequest(ctx)
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	var games []*usecase.GameParam
	for _, gameReq := range req.Games {
		games = append(
			games,
			usecase.NewGameParam(
				gameReq.GoFirst,
				gameReq.WinningFlg,
				gameReq.YourPrizeCards,
				gameReq.OpponentsPrizeCards,
				gameReq.Memo,
			),
		)
	}

	param := usecase.NewMatchParam(
		req.RecordId,
		req.DeckId,
		uid,
		req.OpponentsUserId,
		req.BO3Flg,
		req.QualifyingRoundFlg,
		req.FinalTournamentFlg,
		req.DefaultVictoryFlg,
		req.DefaultDefeatFlg,
		req.VictoryFlg,
		req.OpponentsDeckInfo,
		req.Memo,
		games,
	)

	match, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewMatchUpdateResponse(match)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Match) Delete(ctx *gin.Context) {
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
