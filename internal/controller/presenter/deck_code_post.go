package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func newDeckCodePostUserResponse(user *entity.User, designationTier int) dto.DeckCodePostUserResponse {
	if user == nil {
		return dto.DeckCodePostUserResponse{}
	}

	return dto.DeckCodePostUserResponse{
		ID:              user.ID,
		Name:            user.Name,
		ImageURL:        user.ImageURL,
		DesignationTier: designationTier,
	}
}

func newDeckCodePostResponse(post *entity.DeckCodePost) dto.DeckCodePostResponse {
	sprites := []*dto.PokemonSpriteResponse{}
	for _, s := range post.PokemonSprites {
		sprites = append(sprites, &dto.PokemonSpriteResponse{ID: s.ID, Position: s.Position})
	}

	// いいねした人の称号は一覧では引かない(アイコンだけを重ねて出すため)。
	likers := []dto.DeckCodePostUserResponse{}
	for _, u := range post.RecentLikers {
		likers = append(likers, newDeckCodePostUserResponse(u, 0))
	}

	return dto.DeckCodePostResponse{
		ID:              post.ID,
		PublishedAt:     post.PublishedAt,
		UnpublishedAt:   post.UnpublishedAt,
		Hidden:          !post.HiddenAt.IsZero(),
		User:            newDeckCodePostUserResponse(post.User, post.DesignationTier),
		DeckId:          post.DeckId,
		DeckName:        post.DeckName,
		PokemonSprites:  sprites,
		DeckCodeId:      post.DeckCodeId,
		Code:            post.Code,
		AceSpecCardId:   post.AceSpecCardId,
		AceSpecCardName: post.AceSpecCardName,
		AceSpecImageURL: post.AceSpecImageURL,
		LikeCount:       post.LikeCount,
		LikedByMe:       post.LikedByMe,
		RecentLikers:    likers,
	}
}

func newDeckCodePostResponses(posts []*entity.DeckCodePost) []dto.DeckCodePostResponse {
	ret := []dto.DeckCodePostResponse{}
	for _, post := range posts {
		ret = append(ret, newDeckCodePostResponse(post))
	}

	return ret
}

func newDeckCodePostEnvironmentResponse(env *entity.Environment) *dto.DeckCodePostEnvironmentResponse {
	if env == nil {
		return nil
	}

	return &dto.DeckCodePostEnvironmentResponse{
		ID:       env.ID,
		Title:    env.Title,
		FromDate: env.FromDate,
		ToDate:   env.ToDate,
	}
}

func NewDeckCodePostGetResponse(
	limit int,
	offset int,
	sort string,
	result *usecase.DeckCodePostFindResult,
) *dto.DeckCodePostGetResponse {
	return &dto.DeckCodePostGetResponse{
		Limit:       limit,
		Offset:      offset,
		Sort:        sort,
		Environment: newDeckCodePostEnvironmentResponse(result.Environment),
		Posts:       newDeckCodePostResponses(result.Posts),
	}
}

func NewDeckCodePostGetAceSpecsResponse(result *usecase.DeckCodePostAceSpecCountsResult) *dto.DeckCodePostGetAceSpecsResponse {
	aceSpecs := []dto.DeckCodePostAceSpecCountResponse{}
	for _, a := range result.AceSpecs {
		aceSpecs = append(aceSpecs, dto.DeckCodePostAceSpecCountResponse{
			CardName: a.CardName, ImageURL: a.ImageURL, Count: a.Count,
		})
	}

	return &dto.DeckCodePostGetAceSpecsResponse{
		Environment: newDeckCodePostEnvironmentResponse(result.Environment),
		AceSpecs:    aceSpecs,
	}
}

func NewDeckCodePostGetByIdResponse(post *entity.DeckCodePost) *dto.DeckCodePostGetByIdResponse {
	return &dto.DeckCodePostGetByIdResponse{DeckCodePostResponse: newDeckCodePostResponse(post)}
}

func NewDeckCodePostCreateResponse(post *entity.DeckCodePost) *dto.DeckCodePostCreateResponse {
	return &dto.DeckCodePostCreateResponse{DeckCodePostResponse: newDeckCodePostResponse(post)}
}

func NewDeckCodePostLikeResponse(post *entity.DeckCodePost) *dto.DeckCodePostLikeResponse {
	return &dto.DeckCodePostLikeResponse{DeckCodePostResponse: newDeckCodePostResponse(post)}
}

func NewDeckCodePostGetLikersResponse(
	limit int,
	offset int,
	likers []*entity.DeckCodePostLiker,
) *dto.DeckCodePostGetLikersResponse {
	ret := []dto.DeckCodePostLikerResponse{}
	for _, liker := range likers {
		ret = append(ret, dto.DeckCodePostLikerResponse{
			User:      newDeckCodePostUserResponse(liker.User, liker.DesignationTier),
			CreatedAt: liker.CreatedAt,
		})
	}

	return &dto.DeckCodePostGetLikersResponse{Limit: limit, Offset: offset, Likers: ret}
}

func NewDeckCodePostGetByUserIdResponse(
	limit int,
	offset int,
	view *usecase.DeckCodePostUserView,
) *dto.DeckCodePostGetByUserIdResponse {
	return &dto.DeckCodePostGetByUserIdResponse{
		User:           newDeckCodePostUserResponse(view.User, view.DesignationTier),
		PostCount:      view.Summary.PostCount,
		LikeCountTotal: view.Summary.LikeCountTotal,
		Limit:          limit,
		Offset:         offset,
		Posts:          newDeckCodePostResponses(view.Posts),
	}
}

func NewDeckCodePostGetByDeckIdResponse(posts []*entity.DeckCodePost) *dto.DeckCodePostGetByDeckIdResponse {
	ret := dto.DeckCodePostGetByDeckIdResponse(newDeckCodePostResponses(posts))

	return &ret
}
