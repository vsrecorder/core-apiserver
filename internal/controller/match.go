package controller

import (
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
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	MatchesPath = "/matches"
	// MatchesSummaryPath は複数の記録の対戦集計をまとめて返すエンドポイント。
	MatchesSummaryPath = "/summary"
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
		// "/:id" より前に登録し、summary が記録IDとして解釈されないようにする。
		// 認可はusecase(のリポジトリ)がuidで絞り込むため、authorizationは挟まない。
		r.GET(
			MatchesSummaryPath,
			authentication.RequiredAuthenticationMiddleware(),
			validation.MatchGetSummariesMiddleware(),
			c.GetSummaries,
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
		r.PUT(
			"/:id"+MatchesPath+"/order",
			authentication.RequiredAuthenticationMiddleware(),
			authorization.MatchReorderAuthorizationMiddleware(c.recordRepository),
			validation.MatchReorderMiddleware(),
			c.Reorder,
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
		apierror.ErrBadRequest.JSON(ctx, err)
		return
	}

	matches, err := c.usecase.FindLatest(ctx.Request.Context(), limit)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchResponse{})
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewMatchGetByRecordIdResponse(matches)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	match, err := c.usecase.FindById(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewMatchGetByIdResponse(match)

	ctx.JSON(http.StatusOK, res)
}

func (c *Match) GetByRecordId(ctx *gin.Context) {
	recordId := helper.GetId(ctx)

	matches, err := c.usecase.FindByRecordId(ctx.Request.Context(), recordId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchGetByRecordIdResponse{})
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewMatchGetByRecordIdResponse(matches)

	ctx.JSON(http.StatusOK, res)
}

// GetSummaries は record_ids で指定された記録の対戦集計をまとめて返す。
//
// 記録一覧は1ページぶん(10件)の記録それぞれについて勝敗数を表示するが、
// 必要なのは集計値だけで対戦一覧そのものではない。記録ごとに
// GET /records/:id/matches を叩くと1ページで10往復になるため、ここで1回に束ねる。
//
// 認可はリポジトリのクエリが records.user_id = uid で絞ることで担保する。
// 他人の記録・存在しない記録は404や403にせず結果から除外する。エラーにすると
// 1件でも他人のIDが混ざったページ全体が表示できなくなるうえ、記録IDの
// 実在有無を他人へ知らせることにもなるため。webapp側は欠けた記録だけ
// 個別に取り直す。
func (c *Match) GetSummaries(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	recordIds := helper.GetRecordIds(ctx)

	summaries, err := c.usecase.FindSummariesByRecordIds(ctx.Request.Context(), uid, recordIds)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewMatchGetSummariesResponse(summaries)

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
		apierror.ErrBadRequest.JSON(ctx, err)
		return
	}

	matches, err := c.usecase.FindByUserId(ctx.Request.Context(), userId, limit)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, []*dto.MatchResponse{})
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
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
		pokemonSprites = append(
			pokemonSprites,
			usecase.NewPokemonSpriteParamWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
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
		req.DrawFlg,
		req.GroupMatchVictoryFlg,
		req.OpponentsDeckInfo,
		req.Memo,
		games,
		pokemonSprites,
	)
	// TagIds は NewMatchParam の引数に含めていないため、ここで直接設定する。
	param.TagIds = req.TagIds

	match, err := c.usecase.Create(ctx.Request.Context(), param)
	if err != nil {
		// 対戦結果の整合性エラーは 400。middleware を通れば通常は発生しないが、
		// usecase 層でも検証しているため防御的に 400 を返す。
		if errors.Is(err, apperror.ErrInvalidMatch) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		apierror.ErrInternalServerError.JSON(ctx, err)
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
		pokemonSprites = append(
			pokemonSprites,
			usecase.NewPokemonSpriteParamWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
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
		req.DrawFlg,
		req.GroupMatchVictoryFlg,
		req.OpponentsDeckInfo,
		req.Memo,
		games,
		pokemonSprites,
	)
	// TagIds は NewMatchParam の引数に含めていないため、ここで直接設定する。
	param.TagIds = req.TagIds

	match, err := c.usecase.Update(ctx.Request.Context(), id, param)
	if err != nil {
		// 対戦結果の整合性エラーは 400。
		if errors.Is(err, apperror.ErrInvalidMatch) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewMatchUpdateResponse(match)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Match) Delete(ctx *gin.Context) {
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

func (c *Match) Reorder(ctx *gin.Context) {
	recordId := helper.GetId(ctx)
	req := helper.GetMatchReorderRequest(ctx)

	var orders []*entity.MatchOrder
	for _, m := range req.Matches {
		orders = append(orders, &entity.MatchOrder{
			ID:                 m.Id,
			QualifyingRoundFlg: m.QualifyingRoundFlg,
			FinalTournamentFlg: m.FinalTournamentFlg,
		})
	}

	if err := c.usecase.Reorder(ctx.Request.Context(), recordId, orders); err != nil {
		if err == apperror.ErrInvalidMatchOrder {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
