package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	MatchesPath = "/matches"
)

type Match struct {
	router           *gin.Engine
	matchRepository  repository.MatchInterface
	recordRepository repository.RecordInterface
	usecase          usecase.MatchInterface
}

func NewMatch(
	router *gin.Engine,
	matchRepository repository.MatchInterface,
	recordRepository repository.RecordInterface,
	usecase usecase.MatchInterface,
) *Match {
	return &Match{router, matchRepository, recordRepository, usecase}
}

func (c *Match) RegisterRoute(relativePath string) {
	{
		r := c.router.Group(relativePath + MatchesPath)
		r.GET(
			"",
			authentication.RequiredAuthenticationMiddleware(),
			c.GetLatest,
		)
		r.GET(
			"/:id",
			authentication.OptionalAuthenticationMiddleware(),
			authorization.MatchGetByIdAuthorizationMiddleware(c.matchRepository, c.recordRepository),
			c.GetById,
		)
		r.POST(
			"",
			authentication.RequiredAuthenticationMiddleware(),
			validation.MatchCreateMiddleware(),
			c.Create,
		)
		r.PUT(
			"/:id",
			authentication.RequiredAuthenticationMiddleware(),
			authorization.MatchUpdateAuthorizationMiddleware(c.matchRepository),
			validation.MatchUpdateMiddleware(),
			c.Update,
		)
		r.DELETE(
			"/:id",
			authentication.RequiredAuthenticationMiddleware(),
			authorization.MatchDeleteAuthorizationMiddleware(c.matchRepository),
			c.Delete,
		)
	}

	{
		r := c.router.Group(relativePath + RecordsPath)
		r.GET(
			"/:id"+MatchesPath,
			authentication.OptionalAuthenticationMiddleware(),
			authorization.MatchGetByRecordIdAuthorizationMiddleware(c.recordRepository),
			c.GetByRecordId,
		)
	}

	{
		r := c.router.Group(relativePath + UsersPath)
		r.GET(
			"/:id"+MatchesPath,
			authentication.RequiredAuthenticationMiddleware(),
			c.GetByUserId,
		)
	}
}

func (c *Match) GetLatest(ctx *gin.Context) {
	limit, err := helper.ParseQueryLimit(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}

	matches, err := c.usecase.FindLatest(context.Background(), limit)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchResponse{})
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewMatchGetByRecordIdResponse(matches)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	match, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewMatchGetByIdResponse(match)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) GetByRecordId(ctx *gin.Context) {
	recordId := helper.GetId(ctx)

	matches, err := c.usecase.FindByRecordId(context.Background(), recordId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchGetByRecordIdResponse{})
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewMatchGetByRecordIdResponse(matches)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) GetByUserId(ctx *gin.Context) {
	userId := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	if uid != userId {
		apierror.ErrForbidden.JSON(ctx)
		return
	}

	limit, err := helper.ParseQueryLimit(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}

	matches, err := c.usecase.FindByUserId(context.Background(), userId, limit)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchResponse{})
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewMatchGetByRecordIdResponse(matches)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) Create(ctx *gin.Context) {
	req := helper.GetMatchCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	var pokemonSprites []*usecase.PokemonSpriteParam
	for _, pokemonSprite := range req.PokemonSprites {
		pokemonSprites = append(pokemonSprites, usecase.NewPokemonSpriteParam(pokemonSprite.ID))
	}

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
		req.DeckCodeId,
		uid,
		req.OpponentsUserId,
		req.BO3Flg,
		req.GroupMatchFlg,
		req.QualifyingRoundFlg,
		req.FinalTournamentFlg,
		req.DefaultVictoryFlg,
		req.DefaultDefeatFlg,
		req.VictoryFlg,
		req.GroupMatchVictoryFlg,
		req.OpponentsDeckInfo,
		req.Memo,
		games,
		pokemonSprites,
	)

	match, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewMatchCreateResponse(match)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Match) Update(ctx *gin.Context) {
	req := helper.GetMatchUpdateRequest(ctx)
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	var pokemonSprites []*usecase.PokemonSpriteParam
	for _, pokemonSprite := range req.PokemonSprites {
		pokemonSprites = append(pokemonSprites, usecase.NewPokemonSpriteParam(pokemonSprite.ID))
	}

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
		req.DeckCodeId,
		uid,
		req.OpponentsUserId,
		req.BO3Flg,
		req.GroupMatchFlg,
		req.QualifyingRoundFlg,
		req.FinalTournamentFlg,
		req.DefaultVictoryFlg,
		req.DefaultDefeatFlg,
		req.VictoryFlg,
		req.GroupMatchVictoryFlg,
		req.OpponentsDeckInfo,
		req.Memo,
		games,
		pokemonSprites,
	)

	match, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewMatchUpdateResponse(match)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Match) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(context.Background(), id); err != nil {
		if err == apperror.ErrRecordNotFound {
			apierror.ErrBadRequestNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
