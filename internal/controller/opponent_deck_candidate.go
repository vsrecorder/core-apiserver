package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	OpponentDeckCandidatesPath = "/opponent_deck_candidates"
)

// OpponentDeckCandidate は相手デッキの入力候補(全ユーザーの対戦結果からの集計)を返す。
//
// 自分の対戦がまだ無いユーザーの対戦結果フォームで、相手デッキの表記とスプライトを
// 候補として出すために使う。これまで GET /matches(最新の対戦結果)を流用していた用途を
// 置き換えるもので、対戦結果そのものではなく「表記 × スプライト」の出現回数だけを返す。
// 記録の公開・非公開は問わず集計する(誰の対戦かは返さない)。
type OpponentDeckCandidate struct {
	router  *gin.Engine
	usecase usecase.OpponentDeckCandidateInterface
}

func NewOpponentDeckCandidate(
	router *gin.Engine,
	usecase usecase.OpponentDeckCandidateInterface,
) *OpponentDeckCandidate {
	return &OpponentDeckCandidate{router, usecase}
}

// RegisterRoute は GET /matches/opponent_deck_candidates を登録する。
// 全ユーザーの集計とはいえ、対戦結果を書く画面からしか使わないので認証を要求する。
func (c *OpponentDeckCandidate) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + MatchesPath)
	r.GET(
		OpponentDeckCandidatesPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.OpponentDeckCandidateGetMiddleware(),
		c.Get,
	)
}

func (c *OpponentDeckCandidate) Get(ctx *gin.Context) {
	limit := helper.GetLimit(ctx)

	candidates, err := c.usecase.FindOpponentDeckCandidates(ctx.Request.Context(), limit)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewOpponentDeckCandidatesGetResponse(limit, candidates))
}
