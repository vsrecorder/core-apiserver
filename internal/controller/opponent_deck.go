package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	OpponentDecksPath = "/opponent_decks"
	// OpponentDeckMatchesPath は組み合わせ 1 つの対戦の一覧(/matches/opponent_decks/matches)。
	OpponentDeckMatchesPath = OpponentDecksPath + MatchesPath
)

// OpponentDeck は自分の対戦結果に付けた相手デッキ(表記 × スプライト)の一覧と、一括での置き換え。
//
// 表記ゆれ(「ドラパ」「ドラパルト」)やスプライトの付け忘れを、同じ組み合わせの対戦ごとに
// まとめて直すために使う(adr/opponent-deck-bulk-replace.md)。
// 対象は常に認証したユーザー自身の対戦で、パスやボディで対戦・記録を指定しないため
// 認可ミドルウェアは挟まない(リポジトリが uid で絞る)。
type OpponentDeck struct {
	router  *gin.Engine
	usecase usecase.OpponentDeckInterface
}

func NewOpponentDeck(
	router *gin.Engine,
	usecase usecase.OpponentDeckInterface,
) *OpponentDeck {
	return &OpponentDeck{router, usecase}
}

// RegisterRoute は GET / PUT /matches/opponent_decks と GET /matches/opponent_decks/matches を登録する。
func (c *OpponentDeck) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + MatchesPath)
	r.GET(
		OpponentDecksPath,
		authentication.RequiredAuthenticationMiddleware(),
		c.Get,
	)
	r.GET(
		OpponentDeckMatchesPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.OpponentDeckMatchesGetMiddleware(),
		c.GetMatches,
	)
	r.PUT(
		OpponentDecksPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.OpponentDeckReplaceMiddleware(),
		c.Replace,
	)
}

func (c *OpponentDeck) Get(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	decks, err := c.usecase.FindByUserId(ctx.Request.Context(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewOpponentDecksGetResponse(decks))
}

// GetMatches は組み合わせ 1 つの対戦を、どの記録でどんな結果だったかが分かる形で返す。
// 一括編集で、置き換える前にその表記をどこで使ったかを確かめるために使う。
func (c *OpponentDeck) GetMatches(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	spec := helper.GetOpponentDeckSpecRequest(ctx)

	matches, err := c.usecase.FindMatches(ctx.Request.Context(), uid, opponentDeckSpecParamOf(&spec))
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewOpponentDeckMatchesGetResponse(matches))
}

func (c *OpponentDeck) Replace(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	req := helper.GetOpponentDeckReplaceRequest(ctx)

	param := usecase.NewOpponentDeckReplaceParam(
		opponentDeckSpecParamOf(req.From),
		opponentDeckSpecParamOf(req.To),
	)

	updated, err := c.usecase.Replace(ctx.Request.Context(), uid, param)
	if err != nil {
		// 存在しないスプライトID(外部キー違反)は 400。
		if errors.Is(err, apperror.ErrInvalidReference) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewOpponentDeckReplaceResponse(updated))
}

func opponentDeckSpecParamOf(spec *dto.OpponentDeckSpecRequest) *usecase.OpponentDeckSpecParam {
	sprites := make([]*usecase.PokemonSpriteParam, 0, len(spec.PokemonSprites))
	for _, sprite := range spec.PokemonSprites {
		sprites = append(sprites, usecase.NewPokemonSpriteParamWithPosition(sprite.ID, sprite.Position))
	}

	return usecase.NewOpponentDeckSpecParam(spec.OpponentsDeckInfo, sprites)
}
