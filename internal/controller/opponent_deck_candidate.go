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

// OpponentDeckCandidate は相手デッキの入力候補を返す。
//
// 対戦結果フォームで相手デッキの表記とスプライトを候補として出すために使う。
// 認証したユーザー自身の履歴からの候補を先頭に置き、不足分を全ユーザーの候補で埋める。
// 対戦結果そのものではなく「表記 × スプライト」の出現回数だけを返すため、記録の公開・非公開は
// 問わず集計する(誰の対戦かは返さない)。集計対象外(ignore_stats_flg)の記録は数えない。
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
	uid := helper.GetUID(ctx)

	candidates, err := c.usecase.FindOpponentDeckCandidates(ctx.Request.Context(), uid, limit)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewOpponentDeckCandidatesGetResponse(limit, candidates))
}
