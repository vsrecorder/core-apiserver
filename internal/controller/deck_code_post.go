package controller

import (
	"errors"
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
	DeckCodePostsPath = "/deck_code_posts"
)

// DeckCodePost は「みんなの公開デッキ」(公開したデッキコードの投稿)のAPI。
//
// 一覧・個別・いいねした人・投稿者ページは未ログインでも参照できる(閲覧を公開する方針)。
// 公開・取り下げ・いいねはログインが必要。
type DeckCodePost struct {
	router         *gin.Engine
	repository     repository.DeckCodePostInterface
	deckRepository repository.DeckInterface
	usecase        usecase.DeckCodePostInterface
}

func NewDeckCodePost(
	router *gin.Engine,
	repository repository.DeckCodePostInterface,
	deckRepository repository.DeckInterface,
	usecase usecase.DeckCodePostInterface,
) *DeckCodePost {
	return &DeckCodePost{router, repository, deckRepository, usecase}
}

func (c *DeckCodePost) RegisterRoute(relativePath string) {
	{
		r := c.router.Group(relativePath + DeckCodePostsPath)
		r.GET(
			"",
			authentication.OptionalAuthenticationMiddleware(),
			validation.DeckCodePostGetMiddleware(),
			c.Get,
		)
		r.GET(
			"/:id",
			authentication.OptionalAuthenticationMiddleware(),
			c.GetById,
		)
		r.POST(
			"",
			authentication.RequiredAuthenticationMiddleware(),
			validation.DeckCodePostCreateMiddleware(),
			c.Create,
		)
		r.DELETE(
			"/:id",
			authentication.RequiredAuthenticationMiddleware(),
			authorization.DeckCodePostDeleteAuthorizationMiddleware(c.repository),
			c.Delete,
		)
		r.PUT(
			"/:id/like",
			authentication.RequiredAuthenticationMiddleware(),
			c.Like,
		)
		r.DELETE(
			"/:id/like",
			authentication.RequiredAuthenticationMiddleware(),
			c.Unlike,
		)
		r.GET(
			"/:id/likes",
			authentication.OptionalAuthenticationMiddleware(),
			validation.DeckCodePostPagingMiddleware(),
			c.GetLikers,
		)
		r.POST(
			"/:id/import",
			authentication.RequiredAuthenticationMiddleware(),
			c.Import,
		)
	}

	{
		r := c.router.Group(relativePath + UsersPath)
		r.GET(
			"/:id"+DeckCodePostsPath,
			authentication.OptionalAuthenticationMiddleware(),
			validation.DeckCodePostPagingMiddleware(),
			c.GetByUserId,
		)
	}

	{
		r := c.router.Group(relativePath + DecksPath)
		r.GET(
			"/:id"+DeckCodePostsPath,
			authentication.RequiredAuthenticationMiddleware(),
			authorization.DeckAuthorizationMiddleware(c.deckRepository),
			c.GetByDeckId,
		)
	}
}

func (c *DeckCodePost) Get(ctx *gin.Context) {
	limit := helper.GetLimit(ctx)
	offset := helper.GetOffset(ctx)
	sort := helper.GetSort(ctx)

	param := &usecase.DeckCodePostFindParam{
		Sort:             sort,
		EnvironmentId:    helper.GetEnvironmentId(ctx),
		AceSpecCardId:    helper.GetAceSpecCardId(ctx),
		PokemonSpriteIds: helper.GetPokemonSpriteIds(ctx),
		ViewerUserId:     helper.GetUID(ctx),
		Limit:            limit,
		Offset:           offset,
	}

	result, err := c.usecase.Find(ctx.Request.Context(), param)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostGetResponse(limit, offset, sort, result))
}

func (c *DeckCodePost) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	post, err := c.usecase.FindById(ctx.Request.Context(), id, helper.GetUID(ctx))
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	// 取り下げ済み・非表示は 410 にして、個別ページが「公開を終了しました」を出せるようにする。
	// ただし運営が非表示にした(取り下げてはいない)投稿は、投稿者本人にだけ hidden 付きで返し、
	// 本人が「公開中なのに開けない」と見えないようにする。
	if !post.IsActive() {
		isHiddenOnly := post.UnpublishedAt.IsZero() && !post.HiddenAt.IsZero()
		if !isHiddenOnly || helper.GetUID(ctx) != post.UserId {
			apierror.ErrDeckCodePostUnpublished.JSON(ctx)
			return
		}
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostGetByIdResponse(post))
}

func (c *DeckCodePost) Create(ctx *gin.Context) {
	req := helper.GetDeckCodePostCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	post, err := c.usecase.Publish(ctx.Request.Context(), uid, req.DeckCodeId)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrRecordNotFound):
			apierror.ErrNotFound.JSON(ctx, err)
		case errors.Is(err, apperror.ErrDeckArchived):
			apierror.ErrDeckArchived.JSON(ctx, err)
		case errors.Is(err, apperror.ErrRepublishTooSoon):
			apierror.ErrRepublishTooSoon.JSON(ctx, err)
		case errors.Is(err, apperror.ErrDeckCodePostHidden):
			apierror.ErrDeckCodePostHidden.JSON(ctx, err)
		default:
			apierror.ErrInternalServerError.JSON(ctx, err)
		}
		return
	}

	ctx.JSON(http.StatusCreated, presenter.NewDeckCodePostCreateResponse(post))
}

func (c *DeckCodePost) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Unpublish(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}

func (c *DeckCodePost) Like(ctx *gin.Context) {
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	post, err := c.usecase.Like(ctx.Request.Context(), id, uid)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostLikeResponse(post))
}

func (c *DeckCodePost) Unlike(ctx *gin.Context) {
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	post, err := c.usecase.Unlike(ctx.Request.Context(), id, uid)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostLikeResponse(post))
}

// Import は「取り込む」が使われたことを記録する(同じ人は1回として数える)。取り込み自体
// (デッキの作成)は既存のデッキ作成APIが行い、ここは指標のためだけに呼ばれる。
func (c *DeckCodePost) Import(ctx *gin.Context) {
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	if err := c.usecase.RecordImport(ctx.Request.Context(), id, uid); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}

func (c *DeckCodePost) GetLikers(ctx *gin.Context) {
	id := helper.GetId(ctx)
	limit := helper.GetLimit(ctx)
	offset := helper.GetOffset(ctx)

	likers, err := c.usecase.FindLikers(ctx.Request.Context(), id, limit, offset)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostGetLikersResponse(limit, offset, likers))
}

func (c *DeckCodePost) GetByUserId(ctx *gin.Context) {
	id := helper.GetId(ctx)
	limit := helper.GetLimit(ctx)
	offset := helper.GetOffset(ctx)

	view, err := c.usecase.FindByUserId(ctx.Request.Context(), id, helper.GetUID(ctx), limit, offset)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostGetByUserIdResponse(limit, offset, view))
}

func (c *DeckCodePost) GetByDeckId(ctx *gin.Context) {
	id := helper.GetId(ctx)

	posts, err := c.usecase.FindActiveByDeckId(ctx.Request.Context(), id)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presenter.NewDeckCodePostGetByDeckIdResponse(posts))
}
